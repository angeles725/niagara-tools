package modbus

import (
	"encoding/binary"
	"fmt"
)

// BitsResponse is the response PDU of function codes 1 and 2.
func BitsResponse(function byte, bits []bool) []byte {
	packed := packBits(bits)
	return append([]byte{function, byte(len(packed))}, packed...)
}

// WordsResponse is the response PDU of function codes 3 and 4.
func WordsResponse(function byte, words []uint16) []byte {
	return append([]byte{function, byte(2 * len(words))}, packWords(words)...)
}

// EchoResponse is the response PDU of a write: the request itself for function
// codes 5 and 6, its address and quantity for 15 and 16.
func EchoResponse(r Request) []byte {
	if shapes[r.Function].single {
		return r.Bytes()
	}
	out := []byte{r.Function, byte(r.Address >> 8), byte(r.Address)}
	return binary.BigEndian.AppendUint16(out, r.Quantity)
}

// ExceptionResponse is the exception response PDU to a request of function.
func ExceptionResponse(function byte, e Exception) []byte {
	return []byte{function | exceptionFlag, byte(e)}
}

// Response is a response PDU as a client reads it.
type Response struct {
	Function  byte
	Exception Exception // None unless the device answered an exception
	Bits      []bool
	Words     []uint16
}

// ParseResponse reads the response PDU to req. An exception response is a
// Response with its Exception set; an error means the bytes do not answer req.
func ParseResponse(req Request, pdu []byte) (Response, error) {
	if len(pdu) < 2 {
		return Response{}, fmt.Errorf("modbus: response of %d bytes", len(pdu))
	}
	res := Response{Function: pdu[0] &^ exceptionFlag}
	if res.Function != req.Function {
		return res, fmt.Errorf("modbus: response to function %d, asked %d", res.Function, req.Function)
	}
	if pdu[0]&exceptionFlag != 0 {
		res.Exception = Exception(pdu[1])
		return res, nil
	}
	s := shapes[req.Function]
	if s.single || s.multiple {
		if len(pdu) != 5 {
			return res, fmt.Errorf("modbus: write response of %d bytes, want 5", len(pdu))
		}
		return res, nil
	}
	want := 2 * int(req.Quantity)
	if s.bits {
		want = (int(req.Quantity) + 7) / 8
	}
	if int(pdu[1]) != want || len(pdu) != 2+want {
		return res, fmt.Errorf("modbus: %d data bytes (count field %d), want %d", len(pdu)-2, pdu[1], want)
	}
	if s.bits {
		res.Bits = unpackBits(pdu[2:], int(req.Quantity))
	} else {
		res.Words = unpackWords(pdu[2:])
	}
	return res, nil
}
