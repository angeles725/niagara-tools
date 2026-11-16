package bacserver

import (
	"context"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

var (
	deviceID = bacnet.ObjectID{Type: bacnet.Device, Instance: 1001}
	supply   = bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 1}
	fragile  = bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 2}
	damper   = bacnet.ObjectID{Type: bacnet.AnalogOutput, Instance: 1}
	setpoint = bacnet.ObjectID{Type: bacnet.AnalogValue, Instance: 2}
	status   = bacnet.ObjectID{Type: bacnet.BinaryInput, Instance: 1}
	season   = bacnet.ObjectID{Type: bacnet.MultiStateValue, Instance: 4}
	command  = bacnet.ObjectID{Type: bacnet.BinaryOutput, Instance: 1}
)

func f(v float64) *float64 { return &v }
func at(i uint32) *uint32  { return &i }

func ref(p bacnet.Property) bacnet.PropertyRef { return bacnet.PropertyRef{Property: p} }

// bench is a device that is offline from 10 s to 20 s, with one point that
// fails after one read and holds its responses.
func bench(cov bool, deviceDelay time.Duration) *bacobj.Device {
	return benchAPDU(cov, deviceDelay, 1476)
}

// benchAPDU is bench with a device that sends at most maxAPDU octets.
func benchAPDU(cov bool, deviceDelay time.Duration, maxAPDU int) *bacobj.Device {
	one := 1
	points := model.NewDevice(model.DeviceSpec{Name: "unit-a", Faults: model.Faults{Delay: deviceDelay},
		Offline: []model.Window{{From: 10 * time.Second, To: 20 * time.Second}},
		Points: []model.PointSpec{
			{Name: "supply_temp", Initial: 18.5}, {Name: "damper", Initial: 35}, {Name: "setpoint", Initial: 21, Writable: true},
			{Name: "fan_status", Initial: 1}, {Name: "season", Initial: 3, Writable: true},
			{Name: "fragile", Initial: 9, Faults: model.Faults{FailAfterReads: &one, Delay: 300 * time.Millisecond}},
			{Name: "fan_cmd"}, {Name: "mode", Initial: 2},
		}})
	return bacobj.NewDevice(bacobj.DeviceSpec{Instance: 1001, Name: "unit-a", VendorID: 999, VendorName: "Lab", Model: "sim", MaxAPDU: maxAPDU, COV: cov},
		[]bacobj.ObjectSpec{
			{ID: supply, Point: "supply_temp", Units: 62, COVIncrement: 0.5},
			{ID: damper, Point: "damper", Units: 98, Commandable: true, Min: f(0), Max: f(100)},
			{ID: setpoint, Point: "setpoint", Units: 62, Writable: true},
			{ID: status, Point: "fan_status"},
			{ID: season, Point: "season", StateTexts: []string{"Off", "Heat", "Cool"}, Writable: true},
			{ID: fragile, Point: "fragile", Units: 95},
			{ID: command, Point: "fan_cmd", Commandable: true},
			{ID: bacnet.ObjectID{Type: bacnet.MultiStateInput, Instance: 1}, Point: "mode", StateTexts: []string{"Off", "Heat", "Cool"}},
		}, points)
}

// sim is a server on a loopback UDP port with a clock, a sleep and a log the
// test owns.
type sim struct {
	t      *testing.T
	server *Server
	device *bacobj.Device
	addr   netip.AddrPort

	mu      sync.Mutex
	elapsed time.Duration
	sleeps  []time.Duration
	entries []reqlog.BACnet
	handled chan struct{} // one token per datagram the server dealt with
	seen    int
}

func start(t *testing.T, device *bacobj.Device, adjust func(*Options)) *sim {
	t.Helper()
	s := &sim{t: t, device: device, handled: make(chan struct{}, 4096)}
	opts := Options{
		Device:  device,
		Elapsed: func() time.Duration { s.mu.Lock(); defer s.mu.Unlock(); return s.elapsed },
		Now:     func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
		Sleep: func(_ context.Context, d time.Duration) bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.sleeps = append(s.sleeps, d)
			return true
		},
		Log: func(e reqlog.BACnet) { s.mu.Lock(); defer s.mu.Unlock(); s.entries = append(s.entries, e) },
	}
	opts.handled = func() { s.handled <- struct{}{} }
	if adjust != nil {
		adjust(&opts)
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	s.addr = conn.LocalAddr().(*net.UDPAddr).AddrPort()
	s.server = New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.server.Serve(ctx, conn) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Serve ended with %v after a stop", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("the server did not stop")
		}
	})
	return s
}

func (s *sim) advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elapsed += d
}

func (s *sim) logged() []reqlog.BACnet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]reqlog.BACnet(nil), s.entries...)
}

// last is the newest log entry.
func (s *sim) last() reqlog.BACnet {
	s.t.Helper()
	entries := s.logged()
	if len(entries) == 0 {
		s.t.Fatal("nothing was logged")
	}
	return entries[len(entries)-1]
}

func (s *sim) slept() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.sleeps...)
}

// client is a UDP socket on loopback with a deadline, so a missing answer
// fails the test instead of hanging it.
func (s *sim) client() *net.UDPConn {
	s.t.Helper()
	c, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		s.t.Fatal(err)
	}
	s.t.Cleanup(func() { c.Close() })
	if err := c.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		s.t.Fatal(err)
	}
	return c
}

func local(c *net.UDPConn) netip.AddrPort { return c.LocalAddr().(*net.UDPAddr).AddrPort() }

// raw sends a UDP payload as it is.
func (s *sim) raw(c *net.UDPConn, payload []byte) {
	s.t.Helper()
	if _, err := c.WriteToUDPAddrPort(payload, s.addr); err != nil {
		s.t.Fatal(err)
	}
}

// frame sends a BVLL frame.
func (s *sim) frame(c *net.UDPConn, function byte, data []byte) {
	s.t.Helper()
	s.raw(c, bacnet.BVLL{Function: function, Data: data}.Bytes())
}

// unicast sends an APDU as an Original-Unicast-NPDU on the local network.
func (s *sim) unicast(c *net.UDPConn, apdu []byte) {
	s.t.Helper()
	s.frame(c, bacnet.OriginalUnicast, bacnet.NPDU{ExpectingReply: true, APDU: apdu}.Bytes())
}

// packet is one received frame, decoded as far as it goes.
type packet struct {
	from netip.AddrPort
	bvll bacnet.BVLL
	npdu bacnet.NPDU
	apdu bacnet.APDU
}

// recv reads the next frame the client got.
func (s *sim) recv(c *net.UDPConn) packet {
	s.t.Helper()
	buf := make([]byte, 2048)
	n, from, err := c.ReadFromUDPAddrPort(buf)
	if err != nil {
		s.t.Fatalf("no answer: %v", err)
	}
	p := packet{from: from}
	var ok bool
	if p.bvll, ok = bacnet.ParseBVLL(buf[:n]); !ok {
		s.t.Fatalf("the answer % X is not a BVLL frame", buf[:n])
	}
	data := p.bvll.Data
	switch p.bvll.Function {
	case bacnet.ForwardedNPDU:
		_, data, _ = bacnet.ParseForwarded(data)
	case bacnet.OriginalUnicast, bacnet.OriginalBroadcast:
	default:
		return p
	}
	if p.npdu, ok = bacnet.ParseNPDU(data); !ok {
		s.t.Fatalf("the answer carries no NPDU: % X", data)
	}
	if p.apdu, ok = bacnet.ParseAPDU(p.npdu.APDU); !ok {
		s.t.Fatalf("the answer carries no APDU: % X", p.npdu.APDU)
	}
	return p
}

// ask sends a confirmed request from a client that accepts 1476 octets and
// returns the answer, which must echo the invoke id.
func (s *sim) ask(c *net.UDPConn, invoke, service byte, data []byte) bacnet.APDU {
	s.t.Helper()
	s.unicast(c, bacnet.ConfirmedRequest(bacnet.MaxAPDU, invoke, service, data))
	p := s.recv(c)
	if p.bvll.Function != bacnet.OriginalUnicast || p.from != s.addr || p.apdu.InvokeID != invoke || p.npdu.HasDest {
		s.t.Fatalf("the answer %+v is not a unicast of invoke id %d from the server", p, invoke)
	}
	return p.apdu
}

// settle proves that nothing sent before it was answered: the next frame the
// client gets is the answer to a request made now, and requests are handled in
// order.
func (s *sim) settle(c *net.UDPConn) {
	s.t.Helper()
	const marker = 0xEE
	a := s.ask(c, marker, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: deviceID, Ref: ref(bacnet.PropObjectIdentifier)}.Bytes())
	if a.Type != bacnet.PDUComplexACK {
		s.t.Fatalf("the marker request got %+v", a)
	}
}

// read asks ReadProperty and returns the values, or fails the test.
func (s *sim) read(c *net.UDPConn, object bacnet.ObjectID, r bacnet.PropertyRef) []bacnet.Value {
	s.t.Helper()
	a := s.ask(c, 1, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: object, Ref: r}.Bytes())
	ack, err := bacnet.DecodeReadPropertyAck(a.Data)
	if a.Type != bacnet.PDUComplexACK || a.Service != bacnet.ServiceReadProperty || err != nil || ack.Object != object || ack.Ref.Property != r.Property {
		s.t.Fatalf("read %v %v: %+v, %v", object, r.Property, a, err)
	}
	return ack.Values
}

func (s *sim) write(c *net.UDPConn, object bacnet.ObjectID, priority uint32, value bacnet.Value) bacnet.APDU {
	s.t.Helper()
	q := bacnet.WriteProperty{Object: object, Ref: ref(bacnet.PropPresentValue), Values: []bacnet.Value{value}, HasPriority: priority != 0, Priority: priority}
	return s.ask(c, 2, bacnet.ServiceWriteProperty, q.Bytes())
}

// await blocks until the server has dealt with total datagrams since it
// started, answered or not, so a test can move the clock knowing what was
// handled before.
func (s *sim) await(total int) {
	s.t.Helper()
	timeout := time.After(5 * time.Second)
	for s.seen < total {
		select {
		case <-s.handled:
			s.seen++
		case <-timeout:
			s.t.Fatalf("the server dealt with %d datagrams, want %d", s.seen, total)
		}
	}
}
