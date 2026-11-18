package writeops

import (
	"strconv"
	"strings"
)

// BBacnetAddress addressType values (bacnet-rt BBacnetAddress.MAC_TYPE_*).
const (
	macTypeIP   = "2"
	macTypeMSTP = "3"
)

// MacDisplay is a discovered MAC as Workbench's Device Manager shows it in its "MAC Addr" column:
// an MS/TP station number in decimal ("0f" is 15), a BACnet/IP address as a.b.c.d:port, and any
// other address as the station's lowercase hex octets. kind is the address's addressType ("" when
// the row omits it: one octet is then MS/TP, six are BACnet/IP).
func MacDisplay(kind, mac string) string {
	octets := strings.Fields(mac)
	if len(octets) == 1 && len(octets[0]) > 2 && len(octets[0])%2 == 0 { // "c0a8010abac0"
		octets = nil
		for i := 0; i+2 <= len(mac); i += 2 {
			octets = append(octets, mac[i:i+2])
		}
	}
	bytes := make([]int, len(octets))
	for i, o := range octets {
		n, err := strconv.ParseUint(o, 16, 8)
		if err != nil {
			return mac
		}
		bytes[i] = int(n)
	}
	switch {
	case len(bytes) == 1 && (kind == macTypeMSTP || kind == ""):
		return strconv.Itoa(bytes[0])
	case len(bytes) == 6 && (kind == macTypeIP || kind == ""):
		return strconv.Itoa(bytes[0]) + "." + strconv.Itoa(bytes[1]) + "." + strconv.Itoa(bytes[2]) + "." +
			strconv.Itoa(bytes[3]) + ":" + strconv.Itoa(bytes[4]<<8|bytes[5])
	}
	return mac
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
