package mbserver

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/modbus"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

const bench = `{
  "devices": [
    {"name": "unit-a", "unit": 1, "points": [
      {"name": "run", "table": "coil", "address": 0, "value": 1, "writable": true},
      {"name": "heat", "table": "coil", "address": 1, "value": 0, "writable": true},
      {"name": "locked", "table": "coil", "address": 2, "value": 1},
      {"name": "status", "table": "discrete_input", "address": 0, "value": 1},
      {"name": "alarm", "table": "discrete_input", "address": 1, "value": 0},
      {"name": "setpoint", "table": "holding_register", "address": 0, "type": "uint16", "value": 215, "writable": true},
      {"name": "speed", "table": "holding_register", "address": 1, "type": "float32", "value": 1.5, "writable": true},
      {"name": "serial", "table": "holding_register", "address": 5, "type": "uint16", "value": 777},
      {"name": "temp", "table": "input_register", "address": 0, "type": "int16", "value": -2},
      {"name": "wave", "table": "input_register", "address": 1, "type": "uint16", "value": 100,
       "behaviour": {"kind": "sine", "amplitude": 50, "period_ms": 4000}},
      {"name": "fragile", "table": "input_register", "address": 2, "type": "uint16", "value": 9,
       "faults": {"fail_after_reads": 1, "delay_ms": 300}}
    ]},
    {"name": "unit-b", "unit": 2,
     "faults": {"delay_ms": 150, "offline": [{"from_ms": 10000, "to_ms": 20000}]},
     "points": [{"name": "count", "table": "input_register", "address": 0, "type": "uint16", "value": 5}]}
  ]
}`

// sim is a server on a loopback port with a clock, a sleep and a log the test owns.
type sim struct {
	t      *testing.T
	server *Server
	addr   string
	stop   context.CancelFunc
	done   chan error

	mu      sync.Mutex
	now     time.Time
	sleeps  []time.Duration
	entries []reqlog.Entry
}

var epoch = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func start(t *testing.T, adjust func(*Options)) *sim {
	t.Helper()
	f, faults := scenario.Parse([]byte(bench))
	if len(faults) != 0 {
		t.Fatalf("bench scenario: %v", faults)
	}
	s := &sim{t: t, now: epoch, done: make(chan error, 1)}
	opts := Options{
		Units:          f.Build().Units,
		MaxConnections: 4,
		IdleTimeout:    5 * time.Second,
		Clock:          func() time.Time { s.mu.Lock(); defer s.mu.Unlock(); return s.now },
		Sleep: func(_ context.Context, d time.Duration) bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.sleeps = append(s.sleeps, d)
			return true
		},
		Log: func(e reqlog.Entry) { s.mu.Lock(); defer s.mu.Unlock(); s.entries = append(s.entries, e) },
	}
	if adjust != nil {
		adjust(&opts)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s.addr = ln.Addr().String()
	s.server = New(opts)
	ctx, cancel := context.WithCancel(context.Background())
	s.stop = cancel
	go func() { s.done <- s.server.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-s.done:
		case <-time.After(5 * time.Second):
			t.Error("the server did not stop")
		}
	})
	return s
}

func (s *sim) advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = s.now.Add(d)
}

func (s *sim) logged() []reqlog.Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]reqlog.Entry(nil), s.entries...)
}

func (s *sim) slept() []time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]time.Duration(nil), s.sleeps...)
}

func (s *sim) dial() net.Conn {
	s.t.Helper()
	c, err := net.Dial("tcp", s.addr)
	if err != nil {
		s.t.Fatal(err)
	}
	s.t.Cleanup(func() { c.Close() })
	if err := c.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		s.t.Fatal(err)
	}
	return c
}

func send(t *testing.T, c net.Conn, transaction uint16, unit byte, pdu ...byte) {
	t.Helper()
	if _, err := c.Write(modbus.Frame{Transaction: transaction, Unit: unit, PDU: pdu}.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func receive(t *testing.T, c net.Conn) modbus.Frame {
	t.Helper()
	f, err := modbus.ReadFrame(c)
	if err != nil {
		t.Fatalf("no response: %v", err)
	}
	return f
}

// ask sends one request to a unit and returns the response PDU.
func ask(t *testing.T, c net.Conn, unit byte, pdu ...byte) []byte {
	t.Helper()
	send(t, c, 0x0101, unit, pdu...)
	f := receive(t, c)
	if f.Transaction != 0x0101 || f.Unit != unit || f.Protocol != 0 {
		t.Fatalf("response header %+v does not echo transaction 0x0101 and unit %d", f, unit)
	}
	return f.PDU
}

// closedByServer reports whether the next read finds the connection closed by
// the server (an EOF, or a reset when it closed with bytes of ours unread)
// rather than data or the test's own deadline.
func closedByServer(c net.Conn) bool {
	_, err := c.Read(make([]byte, 1))
	var timeout net.Error
	return err != nil && !(errors.As(err, &timeout) && timeout.Timeout())
}
