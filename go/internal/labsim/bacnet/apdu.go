package bacnet

// The PDU types (clause 20.1).
const (
	PDUConfirmed   = 0
	PDUUnconfirmed = 1
	PDUSimpleACK   = 2
	PDUComplexACK  = 3
	PDUSegmentACK  = 4
	PDUError       = 5
	PDUReject      = 6
	PDUAbort       = 7
)

const (
	flagSegmented = 0x08
	flagMore      = 0x04
	flagServer    = 0x01
)

// apduSizes are the sizes the max-APDU-length-accepted nibble of a confirmed
// request can state, by code.
var apduSizes = []int{50, 128, 206, 480, 1024, 1476}

// MaxAPDU is the largest APDU of BACnet/IP.
const MaxAPDU = 1476

// ValidMaxAPDU reports whether size is one a device can state.
func ValidMaxAPDU(size int) bool {
	for _, s := range apduSizes {
		if s == size {
			return true
		}
	}
	return false
}

var pduNames = map[byte]string{
	PDUConfirmed: "confirmed-request", PDUUnconfirmed: "unconfirmed-request", PDUSimpleACK: "simple-ack",
	PDUComplexACK: "complex-ack", PDUSegmentACK: "segment-ack", PDUError: "error", PDUReject: "reject", PDUAbort: "abort",
}

// PDUName names a PDU type.
func PDUName(pdu byte) string { return name(pduNames, pdu, "pdu") }

// APDU is a decoded application layer header and its service data.
type APDU struct {
	Type        byte
	Segmented   bool // a confirmed request that is one segment of several
	MoreFollows bool
	MaxAPDU     int // a confirmed request: the largest response its sender accepts
	InvokeID    byte
	Service     byte
	Server      bool  // an abort: sent by the server
	Reason      byte  // a reject or an abort
	Err         Error // an error
	Data        []byte
}

// ParseAPDU decodes the header of an APDU.
func ParseAPDU(b []byte) (APDU, bool) {
	if len(b) < 1 {
		return APDU{}, false
	}
	a := APDU{Type: b[0] >> 4}
	need := map[byte]int{PDUConfirmed: 4, PDUUnconfirmed: 2, PDUSimpleACK: 3, PDUComplexACK: 3, PDUSegmentACK: 4, PDUError: 3, PDUReject: 3, PDUAbort: 3}[a.Type]
	if a.Type == PDUConfirmed && b[0]&flagSegmented != 0 {
		a.Segmented, a.MoreFollows, need = true, b[0]&flagMore != 0, 6
	}
	if len(b) < need {
		return APDU{}, false
	}
	rest := b[need:]
	switch a.Type {
	case PDUConfirmed:
		a.MaxAPDU = apduSizes[len(apduSizes)-1]
		if code := int(b[1] & 0x0F); code < len(apduSizes) {
			a.MaxAPDU = apduSizes[code]
		}
		a.InvokeID, a.Service = b[2], b[need-1]
	case PDUUnconfirmed:
		a.Service = b[1]
	case PDUSimpleACK, PDUComplexACK:
		a.InvokeID, a.Service = b[1], b[2]
	case PDUError:
		a.InvokeID, a.Service = b[1], b[2]
		r := NewReader(rest)
		class, err1 := r.Value()
		code, err2 := r.Value()
		if err1 != nil || err2 != nil || class.Kind != KindEnum || code.Kind != KindEnum {
			return APDU{}, false
		}
		a.Err = Error{Class: class.Uint, Code: code.Uint}
	case PDUReject, PDUAbort:
		a.InvokeID, a.Reason, a.Server = b[1], b[2], a.Type == PDUAbort && b[0]&flagServer != 0
	case PDUSegmentACK:
		a.InvokeID = b[1]
	}
	if len(rest) > 0 {
		a.Data = rest
	}
	return a, true
}

// ConfirmedRequest builds an unsegmented confirmed request from a client that
// accepts responses of up to maxAPDU octets and no segmentation.
func ConfirmedRequest(maxAPDU int, invoke, service byte, data []byte) []byte {
	code := byte(0)
	for i, size := range apduSizes {
		if size <= maxAPDU {
			code = byte(i)
		}
	}
	return append([]byte{PDUConfirmed << 4, code, invoke, service}, data...)
}

// UnconfirmedRequest builds an unconfirmed request.
func UnconfirmedRequest(service byte, data []byte) []byte {
	return append([]byte{PDUUnconfirmed << 4, service}, data...)
}

// SimpleACK acknowledges a confirmed request that returns nothing.
func SimpleACK(invoke, service byte) []byte { return []byte{PDUSimpleACK << 4, invoke, service} }

// ComplexACK acknowledges a confirmed request with data.
func ComplexACK(invoke, service byte, data []byte) []byte {
	return append([]byte{PDUComplexACK << 4, invoke, service}, data...)
}

// ErrorPDU refuses a confirmed request with an error class and code.
func ErrorPDU(invoke, service byte, e Error) []byte {
	return Enum(e.Code).Append(Enum(e.Class).Append([]byte{PDUError << 4, invoke, service}))
}

// RejectPDU refuses a confirmed request that is not well formed or not known.
func RejectPDU(invoke byte, reason Reject) []byte {
	return []byte{PDUReject << 4, invoke, byte(reason)}
}

// AbortPDU ends a transaction from the server side.
func AbortPDU(invoke byte, reason Abort) []byte {
	return []byte{PDUAbort<<4 | flagServer, invoke, byte(reason)}
}
