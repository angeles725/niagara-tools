package modbus

import "encoding/binary"

// Request is a parsed request PDU. Quantity is 1 for the single writes; Bits
// holds the coils of function codes 5 and 15, Words the registers of 6 and 16.
type Request struct {
	Function byte
	Address  uint16
	Quantity uint16
	Bits     []bool
	Words    []uint16
}

// ParseRequest reads a request PDU. It answers, in the order of the
// specification's state diagrams, IllegalFunction for a function code outside
// the supported set, IllegalDataValue for a malformed request or a quantity
// outside the limits, and IllegalDataAddress for a range that runs past 65535.
// Whether the addresses exist on a device is the caller's question.
func ParseRequest(pdu []byte) (Request, Exception) {
	if len(pdu) == 0 {
		return Request{}, IllegalFunction
	}
	req := Request{Function: pdu[0]}
	s, ok := shapes[req.Function]
	if !ok {
		return req, IllegalFunction
	}
	if len(pdu) < 5 {
		return req, IllegalDataValue
	}
	req.Address = binary.BigEndian.Uint16(pdu[1:3])
	field := binary.BigEndian.Uint16(pdu[3:5])
	data := pdu[5:]
	if s.single {
		return parseSingle(req, s, field, data)
	}
	req.Quantity = field
	if field == 0 || int(field) > s.maxQuantity {
		return req, IllegalDataValue
	}
	if s.multiple {
		if exc := parseData(&req, s, data); exc != None {
			return req, exc
		}
	} else if len(data) != 0 {
		return req, IllegalDataValue
	}
	if int(req.Address)+int(req.Quantity) > addresses {
		return req, IllegalDataAddress
	}
	return req, None
}

// parseSingle reads the value of function codes 5 and 6.
func parseSingle(req Request, s shape, value uint16, rest []byte) (Request, Exception) {
	req.Quantity = 1
	if len(rest) != 0 {
		return req, IllegalDataValue
	}
	if !s.bits {
		req.Words = []uint16{value}
		return req, None
	}
	if value != coilOn && value != coilOff {
		return req, IllegalDataValue
	}
	req.Bits = []bool{value == coilOn}
	return req, None
}

// parseData reads the byte count and data of function codes 15 and 16.
func parseData(req *Request, s shape, data []byte) Exception {
	want := 2 * int(req.Quantity)
	if s.bits {
		want = (int(req.Quantity) + 7) / 8
	}
	if len(data) != 1+want || int(data[0]) != want {
		return IllegalDataValue
	}
	if s.bits {
		req.Bits = unpackBits(data[1:], int(req.Quantity))
	} else {
		req.Words = unpackWords(data[1:])
	}
	return None
}

// Bytes is the request PDU as a client sends it.
func (r Request) Bytes() []byte {
	out := []byte{r.Function, byte(r.Address >> 8), byte(r.Address)}
	switch s := shapes[r.Function]; {
	case s.single && s.bits:
		value := coilOff
		if len(r.Bits) > 0 && r.Bits[0] {
			value = coilOn
		}
		return binary.BigEndian.AppendUint16(out, value)
	case s.single:
		var value uint16
		if len(r.Words) > 0 {
			value = r.Words[0]
		}
		return binary.BigEndian.AppendUint16(out, value)
	case s.multiple && s.bits:
		packed := packBits(r.Bits)
		out = binary.BigEndian.AppendUint16(out, uint16(len(r.Bits)))
		return append(append(out, byte(len(packed))), packed...)
	case s.multiple:
		out = binary.BigEndian.AppendUint16(out, uint16(len(r.Words)))
		return append(append(out, byte(2*len(r.Words))), packWords(r.Words)...)
	}
	return binary.BigEndian.AppendUint16(out, r.Quantity)
}
