package bacserver

import (
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

func whoIs() []byte { return bacnet.UnconfirmedRequest(bacnet.ServiceWhoIs, nil) }

func code(t *testing.T, p packet) int {
	t.Helper()
	if p.bvll.Function != bacnet.BVLCResult || len(p.bvll.Data) != 2 {
		t.Fatalf("%+v is not a BVLC-Result", p.bvll)
	}
	return int(binary.BigEndian.Uint16(p.bvll.Data))
}

func TestAForeignDeviceRegistersAndGetsTheBroadcasts(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	station, other, stranger := s.client(), s.client(), s.client()
	s.frame(station, bacnet.RegisterForeignDevice, []byte{0, 60})
	if got := code(t, s.recv(station)); got != bacnet.ResultSuccess {
		t.Fatalf("registration result %#x", got)
	}
	if e := s.last(); e.Link != "register-foreign-device" || e.Result != reqlog.ResultOK || e.Detail != "ttl 60 s" {
		t.Errorf("log entry %+v", e)
	}
	s.frame(other, bacnet.RegisterForeignDevice, []byte{0, 30})
	s.recv(other)
	s.advance(10 * time.Second) // the device is offline from 10 s: come back at 20 s
	s.advance(10 * time.Second)
	s.frame(station, bacnet.ReadFDT, nil)
	table := s.recv(station)
	entries, ok := bacnet.ParseFDT(table.bvll.Data)
	if table.bvll.Function != bacnet.ReadFDTAck || !ok || len(entries) != 2 {
		t.Fatalf("foreign device table %+v, %v", table.bvll, ok)
	}
	for _, e := range entries {
		want := bacnet.FDTEntry{Address: local(station), TTL: 60, Remaining: 70} // 60 s and 30 s of grace, 20 s gone
		if e.Address == local(other) {
			want = bacnet.FDTEntry{Address: local(other), TTL: 30, Remaining: 40}
		}
		if e != want {
			t.Errorf("table entry %+v, want %+v", e, want)
		}
	}
	// A broadcast the station asks to distribute reaches the other foreign
	// device, and the I-Am it provokes reaches both, as forwarded NPDUs.
	broadcast := bacnet.NPDU{HasDest: true, DNet: bacnet.GlobalBroadcast, Hops: 255, APDU: whoIs()}.Bytes()
	s.frame(station, bacnet.DistributeBroadcast, broadcast)
	relayed := s.recv(other)
	if origin, _, _ := bacnet.ParseForwarded(relayed.bvll.Data); relayed.bvll.Function != bacnet.ForwardedNPDU || origin != local(station) || relayed.apdu.Service != bacnet.ServiceWhoIs {
		t.Errorf("the other foreign device got %+v from %v, want the Who-Is forwarded from the station", relayed.bvll, origin)
	}
	for name, c := range map[string]packet{"the station": s.recv(station), "the other foreign device": s.recv(other)} {
		origin, _, _ := bacnet.ParseForwarded(c.bvll.Data)
		if c.bvll.Function != bacnet.ForwardedNPDU || origin != s.addr || c.apdu.Service != bacnet.ServiceIAm || !c.npdu.HasDest || c.npdu.DNet != bacnet.GlobalBroadcast {
			t.Errorf("%s got %+v (origin %v), want the I-Am as a forwarded global broadcast from the simulator", name, c, origin)
		}
	}
	// A device that never registered may not distribute, and gets its I-Am alone.
	s.frame(stranger, bacnet.DistributeBroadcast, broadcast)
	if got := code(t, s.recv(stranger)); got != bacnet.ResultDistributeNAK {
		t.Errorf("a distribution asked by a stranger: result %#x, want the NAK", got)
	}
	s.advance(41 * time.Second) // past the 30 s registration and its grace, not past the 60 s one
	s.unicast(stranger, whoIs())
	if p := s.recv(stranger); p.bvll.Function != bacnet.OriginalUnicast || p.apdu.Service != bacnet.ServiceIAm {
		t.Errorf("a stranger's Who-Is: %+v, want a unicast I-Am", p)
	}
	if p := s.recv(station); p.bvll.Function != bacnet.ForwardedNPDU || p.apdu.Service != bacnet.ServiceIAm {
		t.Errorf("the registered station did not get the I-Am a stranger provoked: %+v", p)
	}
	s.frame(other, bacnet.ReadFDT, nil)
	if entries, _ := bacnet.ParseFDT(s.recv(other).bvll.Data); len(entries) != 1 || entries[0].Address != local(station) {
		t.Errorf("after the short registration ran out the table is %+v", entries)
	}
	if got := s.server.Stats().ForeignDevices; got != 1 {
		t.Errorf("Stats().ForeignDevices = %d, want 1", got)
	}
	s.frame(other, bacnet.DeleteFDTEntry, bacnet.AppendAddress(nil, local(station)))
	if got := code(t, s.recv(other)); got != bacnet.ResultSuccess {
		t.Errorf("deleting an entry: result %#x", got)
	}
	s.frame(other, bacnet.DeleteFDTEntry, bacnet.AppendAddress(nil, local(station)))
	if got := code(t, s.recv(other)); got != bacnet.ResultDeleteFDTNAK {
		t.Errorf("deleting it again: result %#x, want the NAK", got)
	}
}

func TestTheOtherBVLLFunctionsAreAnsweredOrIgnored(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), func(o *Options) { o.MaxForeignDevices = 1 })
	c, second := s.client(), s.client()
	s.frame(c, bacnet.ReadBDT, nil)
	if p := s.recv(c); p.bvll.Function != bacnet.ReadBDTAck || len(p.bvll.Data) != 0 {
		t.Errorf("the broadcast distribution table: %+v, want an empty one", p.bvll)
	}
	s.frame(c, bacnet.WriteBDT, make([]byte, 10))
	if got := code(t, s.recv(c)); got != bacnet.ResultWriteBDTNAK {
		t.Errorf("writing the table: result %#x, want the NAK", got)
	}
	s.frame(c, bacnet.RegisterForeignDevice, []byte{60}) // half a time to live
	if got := code(t, s.recv(c)); got != bacnet.ResultRegisterNAK {
		t.Errorf("a registration with half a time to live: result %#x, want the NAK", got)
	}
	s.frame(c, bacnet.RegisterForeignDevice, []byte{0, 60, 0}) // and one octet too many
	if got := code(t, s.recv(c)); got != bacnet.ResultRegisterNAK {
		t.Errorf("a registration with three octets of time to live: result %#x, want the NAK", got)
	}
	s.frame(c, bacnet.RegisterForeignDevice, []byte{0, 60})
	s.recv(c)
	s.frame(c, bacnet.RegisterForeignDevice, []byte{0, 90}) // a renewal takes no second entry
	if got := code(t, s.recv(c)); got != bacnet.ResultSuccess {
		t.Errorf("a renewal: result %#x", got)
	}
	s.frame(second, bacnet.RegisterForeignDevice, []byte{0, 60})
	if got := code(t, s.recv(second)); got != bacnet.ResultRegisterNAK {
		t.Errorf("a registration into a full table: result %#x, want the NAK", got)
	}
	// Frames a device does not answer: a forwarded NPDU (no BBMD is a peer), a
	// result, a function that does not exist, and what is not BVLL at all.
	s.frame(c, bacnet.ForwardedNPDU, append(bacnet.AppendAddress(nil, netip.MustParseAddrPort("192.0.2.9:47808")), bacnet.NPDU{APDU: whoIs()}.Bytes()...))
	s.frame(c, bacnet.BVLCResult, []byte{0, 0})
	s.frame(c, 0x7F, nil)
	s.raw(c, []byte("GET / HTTP/1.1\r\n\r\n"))
	s.raw(c, []byte{0x81, 0x0A, 0x00, 0x09, 0x01})
	s.settle(c)
	if got := s.server.Stats(); got.Malformed != 2 {
		t.Errorf("Stats() = %+v, want 2 malformed packets", got)
	}
}

func TestTheNetworkLayerIsLocalOnly(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	s.frame(c, bacnet.OriginalBroadcast, bacnet.NPDU{HasDest: true, DNet: bacnet.GlobalBroadcast, Hops: 255, APDU: whoIs()}.Bytes())
	if p := s.recv(c); p.apdu.Service != bacnet.ServiceIAm {
		t.Errorf("a global broadcast Who-Is: %+v", p)
	}
	s.frame(c, bacnet.OriginalBroadcast, bacnet.NPDU{HasDest: true, DNet: 7, Hops: 255, APDU: whoIs()}.Bytes())
	s.frame(c, bacnet.OriginalBroadcast, bacnet.NPDU{Network: true, MessageType: 0}.Bytes()) // Who-Is-Router-To-Network
	s.frame(c, bacnet.OriginalUnicast, []byte{0x02, 0x00, 0x10, 0x08})                       // another protocol version
	s.frame(c, bacnet.OriginalUnicast, []byte{0x01, 0x00})                                   // no APDU
	s.frame(c, bacnet.OriginalUnicast, bacnet.NPDU{APDU: bacnet.SimpleACK(1, bacnet.ServiceConfirmedCOVNotification)}.Bytes())
	s.settle(c)
	entries := s.logged()
	want := []string{"other_network", "network_layer_message", "malformed_npdu", "malformed_apdu", "not_a_request"}
	for i, reason := range want {
		if e := entries[len(entries)-1-len(want)+i]; e.Result != reqlog.ResultIgnored || e.Reason != reason {
			t.Errorf("entry %d: %+v, want ignored for %s", i, e, reason)
		}
	}
	// A request that came through a router is answered back to its source network.
	routed := bacnet.NPDU{ExpectingReply: true, HasSource: true, SNet: 5, SAdr: []byte{0x0A},
		APDU: bacnet.ConfirmedRequest(bacnet.MaxAPDU, 3, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: supply, Ref: ref(bacnet.PropPresentValue)}.Bytes())}
	s.frame(c, bacnet.OriginalUnicast, routed.Bytes())
	if p := s.recv(c); p.apdu.Type != bacnet.PDUComplexACK || !p.npdu.HasDest || p.npdu.DNet != 5 || len(p.npdu.DAdr) != 1 || p.npdu.DAdr[0] != 0x0A || p.npdu.Hops != 255 {
		t.Errorf("the answer to a routed request: NPDU %+v, want it addressed to network 5, station 0A", p.npdu)
	}
}

func TestAnIAmAlsoGoesToAnExplicitBroadcastAddress(t *testing.T) {
	t.Parallel()
	// Another loopback socket stands for the subnet's broadcast address.
	subnet, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer subnet.Close()
	if err := subnet.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	s := start(t, bench(false, 0), func(o *Options) { o.Broadcast = local(subnet) })
	c := s.client()
	s.unicast(c, whoIs())
	if p := s.recv(c); p.bvll.Function != bacnet.OriginalUnicast || p.apdu.Service != bacnet.ServiceIAm {
		t.Errorf("the asker got %+v", p)
	}
	if p := s.recv(subnet); p.bvll.Function != bacnet.OriginalBroadcast || p.apdu.Service != bacnet.ServiceIAm || p.npdu.DNet != bacnet.GlobalBroadcast {
		t.Errorf("the broadcast address got %+v, want the I-Am as an original broadcast", p)
	}
	if e := s.last(); len(e.To) != 2 {
		t.Errorf("the log names %v, want the asker and the broadcast address", e.To)
	}
}
