package bacnet

import (
	"encoding/binary"
	"net/netip"
)

// bvllType is the first octet of every BACnet/IP frame.
const bvllType = 0x81

// The BVLL functions (Annex J).
const (
	BVLCResult            = 0x00
	WriteBDT              = 0x01
	ReadBDT               = 0x02
	ReadBDTAck            = 0x03
	ForwardedNPDU         = 0x04
	RegisterForeignDevice = 0x05
	ReadFDT               = 0x06
	ReadFDTAck            = 0x07
	DeleteFDTEntry        = 0x08
	DistributeBroadcast   = 0x09
	OriginalUnicast       = 0x0A
	OriginalBroadcast     = 0x0B
	bvllHeader            = 4
	addressSize           = 6  // an IPv4 address and a UDP port
	fdtEntrySize          = 10 // an address, the time to live and the time remaining
)

// The codes of a BVLC-Result.
const (
	ResultSuccess       = 0x0000
	ResultWriteBDTNAK   = 0x0010
	ResultReadBDTNAK    = 0x0020
	ResultRegisterNAK   = 0x0030
	ResultReadFDTNAK    = 0x0040
	ResultDeleteFDTNAK  = 0x0050
	ResultDistributeNAK = 0x0060
)

var bvllNames = map[byte]string{
	BVLCResult: "result", WriteBDT: "write-broadcast-distribution-table", ReadBDT: "read-broadcast-distribution-table",
	ReadBDTAck: "read-broadcast-distribution-table-ack", ForwardedNPDU: "forwarded-npdu",
	RegisterForeignDevice: "register-foreign-device", ReadFDT: "read-foreign-device-table",
	ReadFDTAck: "read-foreign-device-table-ack", DeleteFDTEntry: "delete-foreign-device-table-entry",
	DistributeBroadcast: "distribute-broadcast-to-network", OriginalUnicast: "original-unicast-npdu",
	OriginalBroadcast: "original-broadcast-npdu",
}

// BVLLName names a BVLL function.
func BVLLName(function byte) string { return name(bvllNames, function, "bvll") }

// BVLL is one BACnet/IP frame: a function and what follows its 4-octet header.
type BVLL struct {
	Function byte
	Data     []byte
}

// ParseBVLL decodes a UDP payload; its length field must be the payload's.
func ParseBVLL(p []byte) (BVLL, bool) {
	if len(p) < bvllHeader || p[0] != bvllType || int(binary.BigEndian.Uint16(p[2:])) != len(p) {
		return BVLL{}, false
	}
	return BVLL{Function: p[1], Data: p[bvllHeader:]}, true
}

// Bytes encodes the frame.
func (f BVLL) Bytes() []byte {
	out := []byte{bvllType, f.Function, 0, 0}
	out = append(out, f.Data...)
	binary.BigEndian.PutUint16(out[2:], uint16(len(out)))
	return out
}

// ResultFrame is a BVLC-Result with the given code.
func ResultFrame(code uint16) BVLL {
	return BVLL{Function: BVLCResult, Data: binary.BigEndian.AppendUint16(nil, code)}
}

// RegisterForeignDeviceFrame asks a BBMD for broadcasts for ttl seconds.
func RegisterForeignDeviceFrame(ttl uint16) BVLL {
	return BVLL{Function: RegisterForeignDevice, Data: binary.BigEndian.AppendUint16(nil, ttl)}
}

// AppendAddress writes a B/IP address: 4 octets of IPv4 and the port. An
// address that is not IPv4 writes nothing.
func AppendAddress(b []byte, addr netip.AddrPort) []byte {
	ip := addr.Addr().Unmap()
	if !ip.Is4() {
		return b
	}
	four := ip.As4()
	return binary.BigEndian.AppendUint16(append(b, four[:]...), addr.Port())
}

func parseAddress(b []byte) netip.AddrPort {
	return netip.AddrPortFrom(netip.AddrFrom4([4]byte(b[:4])), binary.BigEndian.Uint16(b[4:]))
}

// ForwardedFrame is an NPDU a BBMD passes on, with the address it came from.
func ForwardedFrame(origin netip.AddrPort, npdu []byte) BVLL {
	return BVLL{Function: ForwardedNPDU, Data: append(AppendAddress(nil, origin), npdu...)}
}

// ParseForwarded splits the data of a Forwarded-NPDU.
func ParseForwarded(data []byte) (netip.AddrPort, []byte, bool) {
	if len(data) < addressSize {
		return netip.AddrPort{}, nil, false
	}
	return parseAddress(data), data[addressSize:], true
}

// FDTEntry is one registered foreign device.
type FDTEntry struct {
	Address   netip.AddrPort
	TTL       uint16 // seconds asked at registration
	Remaining uint16 // seconds before the entry is dropped
}

// ReadFDTAckFrame is the foreign device table as a BBMD reports it.
func ReadFDTAckFrame(entries []FDTEntry) BVLL {
	var data []byte
	for _, e := range entries {
		data = AppendAddress(data, e.Address)
		data = binary.BigEndian.AppendUint16(data, e.TTL)
		data = binary.BigEndian.AppendUint16(data, e.Remaining)
	}
	return BVLL{Function: ReadFDTAck, Data: data}
}

// ParseFDT decodes the data of a Read-Foreign-Device-Table-Ack.
func ParseFDT(data []byte) ([]FDTEntry, bool) {
	if len(data)%fdtEntrySize != 0 {
		return nil, false
	}
	var entries []FDTEntry
	for ; len(data) > 0; data = data[fdtEntrySize:] {
		entries = append(entries, FDTEntry{Address: parseAddress(data),
			TTL: binary.BigEndian.Uint16(data[6:]), Remaining: binary.BigEndian.Uint16(data[8:])})
	}
	return entries, true
}
