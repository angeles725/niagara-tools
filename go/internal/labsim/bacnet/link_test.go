package bacnet

import (
	"bytes"
	"net/netip"
	"reflect"
	"testing"
)

func TestBVLLFramesEncodeAndDecode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		frame BVLL
		wire  string
	}{
		{"an original broadcast", BVLL{Function: OriginalBroadcast, Data: []byte{0x01, 0x20, 0xFF, 0xFF, 0x00, 0xFF, 0x10, 0x08}},
			"81 0B 00 0C 01 20 FF FF 00 FF 10 08"},
		{"an original unicast", BVLL{Function: OriginalUnicast, Data: []byte{0x01, 0x04, 0x00, 0x00, 0x01, 0x0C, 0x0C, 0x00, 0x00, 0x00, 0x05, 0x19, 0x55}},
			"81 0A 00 11 01 04 00 00 01 0C 0C 00 00 00 05 19 55"},
		{"a registration for 60 s", RegisterForeignDeviceFrame(60), "81 05 00 06 00 3C"},
		{"a success result", ResultFrame(ResultSuccess), "81 00 00 06 00 00"},
		{"a refused registration", ResultFrame(ResultRegisterNAK), "81 00 00 06 00 30"},
		{"a forwarded NPDU", ForwardedFrame(netip.MustParseAddrPort("192.0.2.7:47808"), []byte{0x01, 0x00, 0x10, 0x08}),
			"81 04 00 0E C0 00 02 07 BA C0 01 00 10 08"},
		{"an empty broadcast distribution table", BVLL{Function: ReadBDTAck}, "81 03 00 04"},
		{"a foreign device table", ReadFDTAckFrame([]FDTEntry{{Address: netip.MustParseAddrPort("127.0.0.1:47808"), TTL: 60, Remaining: 85}}),
			"81 07 00 0E 7F 00 00 01 BA C0 00 3C 00 55"},
	}
	for _, c := range cases {
		wire := unhex(t, c.wire)
		if got := c.frame.Bytes(); !bytes.Equal(got, wire) {
			t.Errorf("%s: encoded % X, want % X", c.name, got, wire)
		}
		got, ok := ParseBVLL(wire)
		if !ok || got.Function != c.frame.Function || !bytes.Equal(got.Data, c.frame.Data) {
			t.Errorf("%s: decoded %+v, %v", c.name, got, ok)
		}
	}
}

func TestBVLLRefusesWhatIsNotAFrame(t *testing.T) {
	t.Parallel()
	for name, wire := range map[string]string{
		"nothing":                         "",
		"a short header":                  "81 0A 00",
		"another type than BACnet/IP":     "82 0A 00 04",
		"a length longer than the packet": "81 0A 00 08 01 00",
		"a length shorter than it":        "81 0A 00 04 01 00",
		"a length below the header":       "81 0A 00 03",
	} {
		if got, ok := ParseBVLL(unhex(t, wire)); ok {
			t.Errorf("%s was accepted as %+v", name, got)
		}
	}
}

func TestBVLLAddressesAndTables(t *testing.T) {
	t.Parallel()
	addr, npdu, ok := ParseForwarded(unhex(t, "C0 00 02 07 BA C0 01 00"))
	if !ok || addr != netip.MustParseAddrPort("192.0.2.7:47808") || !bytes.Equal(npdu, []byte{1, 0}) {
		t.Errorf("forwarded = %v, % X, %v", addr, npdu, ok)
	}
	if _, _, ok := ParseForwarded([]byte{1, 2, 3, 4, 5}); ok {
		t.Error("a forwarded NPDU with half an address was accepted")
	}
	entries, ok := ParseFDT(unhex(t, "7F 00 00 01 BA C0 00 3C 00 55"))
	if !ok || len(entries) != 1 || entries[0].TTL != 60 || entries[0].Remaining != 85 {
		t.Errorf("table = %+v, %v", entries, ok)
	}
	if _, ok := ParseFDT(make([]byte, 11)); ok {
		t.Error("a table with a partial entry was accepted")
	}
	if got := AppendAddress(nil, netip.MustParseAddrPort("[::1]:47808")); got != nil {
		t.Errorf("an IPv6 address was written as % X: BACnet/IP is IPv4", got)
	}
}

func TestNPDUEncodesAndDecodes(t *testing.T) {
	t.Parallel()
	apdu := []byte{0x10, 0x08}
	cases := []struct {
		name string
		npdu NPDU
		wire string
	}{
		{"local", NPDU{APDU: apdu}, "01 00 10 08"},
		{"local, expecting a reply", NPDU{ExpectingReply: true, APDU: apdu}, "01 04 10 08"},
		{"a global broadcast", NPDU{HasDest: true, DNet: 0xFFFF, Hops: 255, APDU: apdu}, "01 20 FF FF 00 FF 10 08"},
		{"from a routed source", NPDU{HasSource: true, SNet: 5, SAdr: []byte{0x0A}, APDU: apdu}, "01 08 00 05 01 0A 10 08"},
		{"to a routed destination from a routed source", NPDU{HasDest: true, DNet: 7, DAdr: []byte{1, 2}, Hops: 254,
			HasSource: true, SNet: 5, SAdr: []byte{0x0A}, Priority: 1, APDU: apdu}, "01 29 00 07 02 01 02 00 05 01 0A FE 10 08"},
		{"a network layer message", NPDU{Network: true, MessageType: 0, APDU: nil}, "01 80 00"},
	}
	for _, c := range cases {
		wire := unhex(t, c.wire)
		if got := c.npdu.Bytes(); !bytes.Equal(got, wire) {
			t.Errorf("%s: encoded % X, want % X", c.name, got, wire)
		}
		got, ok := ParseNPDU(wire)
		if !ok || !reflect.DeepEqual(got, c.npdu) {
			t.Errorf("%s: decoded %+v, %v; want %+v", c.name, got, ok, c.npdu)
		}
	}
	for name, wire := range map[string]string{
		"nothing": "", "another version": "02 00 10 08", "no control octet": "01",
		"a destination cut short": "01 20 FF", "a destination address cut short": "01 20 00 07 02 01",
		"a source cut short": "01 08 00 05 01", "no hop count": "01 20 FF FF 00",
		"a network message with no type": "01 80", "a source address of length 0": "01 08 00 05 00 10 08",
	} {
		if got, ok := ParseNPDU(unhex(t, wire)); ok {
			t.Errorf("%s was accepted as %+v", name, got)
		}
	}
}
