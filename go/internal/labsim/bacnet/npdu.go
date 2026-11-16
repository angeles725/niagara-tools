package bacnet

import "encoding/binary"

// The NPDU header (clause 6.2).
const (
	npduVersion     = 1
	ctlNetwork      = 0x80 // a network layer message, no APDU
	ctlDest         = 0x20
	ctlSource       = 0x08
	ctlExpecting    = 0x04
	ctlPriority     = 0x03
	GlobalBroadcast = 0xFFFF
	DefaultHopCount = 255
	firstVendorMsg  = 0x80 // message types from here carry a vendor identifier
)

// NPDU is a network layer header and the APDU it carries.
type NPDU struct {
	Network        bool // a network layer message: MessageType is set and APDU is its data
	ExpectingReply bool
	Priority       byte
	HasDest        bool
	DNet           uint16
	DAdr           []byte // empty: a broadcast on DNet
	Hops           byte
	HasSource      bool
	SNet           uint16
	SAdr           []byte
	MessageType    byte
	APDU           []byte
}

// station reads a network number and an address; a source address must have
// at least one octet.
func station(b []byte, source bool) (net uint16, adr, rest []byte, ok bool) {
	if len(b) < 3 || len(b) < 3+int(b[2]) || (source && b[2] == 0) {
		return 0, nil, nil, false
	}
	size := int(b[2])
	if size > 0 {
		adr = append([]byte(nil), b[3:3+size]...)
	}
	return binary.BigEndian.Uint16(b), adr, b[3+size:], true
}

// ParseNPDU decodes a network layer header.
func ParseNPDU(b []byte) (NPDU, bool) {
	if len(b) < 2 || b[0] != npduVersion {
		return NPDU{}, false
	}
	ctl, rest := b[1], b[2:]
	n := NPDU{Network: ctl&ctlNetwork != 0, ExpectingReply: ctl&ctlExpecting != 0, Priority: ctl & ctlPriority,
		HasDest: ctl&ctlDest != 0, HasSource: ctl&ctlSource != 0}
	ok := true
	if n.HasDest {
		if n.DNet, n.DAdr, rest, ok = station(rest, false); !ok {
			return NPDU{}, false
		}
	}
	if n.HasSource {
		if n.SNet, n.SAdr, rest, ok = station(rest, true); !ok {
			return NPDU{}, false
		}
	}
	if n.HasDest {
		if len(rest) < 1 {
			return NPDU{}, false
		}
		n.Hops, rest = rest[0], rest[1:]
	}
	if n.Network {
		if len(rest) < 1 || (rest[0] >= firstVendorMsg && len(rest) < 3) {
			return NPDU{}, false
		}
		n.MessageType, rest = rest[0], rest[1:]
	}
	if len(rest) > 0 {
		n.APDU = rest
	}
	return n, true
}

// Bytes encodes the header and the APDU.
func (n NPDU) Bytes() []byte {
	ctl := n.Priority & ctlPriority
	for flag, on := range map[byte]bool{ctlNetwork: n.Network, ctlDest: n.HasDest, ctlSource: n.HasSource, ctlExpecting: n.ExpectingReply} {
		if on {
			ctl |= flag
		}
	}
	out := []byte{npduVersion, ctl}
	if n.HasDest {
		out = append(binary.BigEndian.AppendUint16(out, n.DNet), byte(len(n.DAdr)))
		out = append(out, n.DAdr...)
	}
	if n.HasSource {
		out = append(binary.BigEndian.AppendUint16(out, n.SNet), byte(len(n.SAdr)))
		out = append(out, n.SAdr...)
	}
	if n.HasDest {
		out = append(out, n.Hops)
	}
	if n.Network {
		out = append(out, n.MessageType)
	}
	return append(out, n.APDU...)
}

// Reply is the header of an answer to n: local, or back to the routed source
// the request came from.
func (n NPDU) Reply(apdu []byte) NPDU {
	out := NPDU{APDU: apdu}
	if n.HasSource {
		out.HasDest, out.DNet, out.DAdr, out.Hops = true, n.SNet, n.SAdr, DefaultHopCount
	}
	return out
}
