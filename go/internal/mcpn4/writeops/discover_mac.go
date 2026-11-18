package writeops

import (
	"strconv"
	"strings"
)

// BBacnetAddress addressType values (bacnet-rt BBacnetAddress.MAC_TYPE_*).
const (
	macTypeEthernet = "1"
	macTypeIP       = "2"
)

// MacDisplay is a discovered MAC as Workbench's Device Manager shows it in its "MAC Addr" column,
// which calls BBacnetAddress.bytesToString (bacnet-rt :217): an Ethernet MAC as colon-separated
// upper-case hex, a BACnet/IP address as a.b.c.d with its port in hex (:0xBAC0), any other address
// as one byte in decimal (an MS/TP station: "0f" is 15) or upper-case hex octets. kind is the
// address's addressType ("" when the row omits it: six octets are then BACnet/IP). A MAC that is
// not hex octets, or an Ethernet MAC that is not six octets, is returned unchanged.
func MacDisplay(kind, mac string) string {
	bytes, ok := macOctets(mac)
	if !ok {
		return mac
	}
	if kind == "" && len(bytes) == 6 {
		kind = macTypeIP
	}
	hex := make([]string, len(bytes))
	for i, b := range bytes {
		hex[i] = strings.ToUpper(strconv.FormatInt(int64(b)|0x100, 16)[1:])
	}
	switch {
	case kind == macTypeEthernet:
		if len(bytes) != 6 {
			return mac
		}
		return strings.Join(hex, ":")
	case kind == macTypeIP && len(bytes) >= 4:
		ip := strconv.Itoa(bytes[0]) + "." + strconv.Itoa(bytes[1]) + "." + strconv.Itoa(bytes[2]) + "." + strconv.Itoa(bytes[3])
		if len(bytes) > 5 {
			ip += ":0x" + strings.ToUpper(strconv.FormatInt(int64(bytes[4]<<8|bytes[5]), 16))
		}
		return ip
	case len(bytes) == 1:
		return strconv.Itoa(bytes[0])
	}
	return strings.Join(hex, " ")
}

// macOctets parses "c0 a8 01 0a" or "c0a8010a" into bytes.
func macOctets(mac string) ([]int, bool) {
	octets := strings.Fields(mac)
	if len(octets) == 1 && len(octets[0]) > 2 && len(octets[0])%2 == 0 { // "c0a8010abac0"
		octets = nil
		for i := 0; i+2 <= len(mac); i += 2 {
			octets = append(octets, mac[i:i+2])
		}
	}
	if len(octets) == 0 {
		return nil, false
	}
	bytes := make([]int, len(octets))
	for i, o := range octets {
		n, err := strconv.ParseUint(o, 16, 8)
		if err != nil {
			return nil, false
		}
		bytes[i] = int(n)
	}
	return bytes, true
}

// objectCount is a discovered device's listSize (its objectList length, Workbench's "Objects"
// column), nil when the station could not read it (-1, the default the row then omits).
func objectCount(v any) any {
	n, ok := wholeNumberText(v)
	if !ok || n < 0 {
		return nil
	}
	return n
}
