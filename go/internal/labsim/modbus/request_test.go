package modbus

import (
	"bytes"
	"reflect"
	"testing"
)

// The request and response bytes below are the example frames of the Modbus
// Application Protocol specification V1.1b3, section 6 (one per function code).

func bitsOf(s string) []bool {
	out := make([]bool, len(s))
	for i := range s {
		out[i] = s[i] == '1'
	}
	return out
}

func TestParseRequestReadsTheSpecificationExamples(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		pdu  []byte
		want Request
	}{
		{"read coils", []byte{0x01, 0x00, 0x13, 0x00, 0x13}, Request{Function: ReadCoils, Address: 0x13, Quantity: 0x13}},
		{"read discrete inputs", []byte{0x02, 0x00, 0xC4, 0x00, 0x16}, Request{Function: ReadDiscreteInputs, Address: 0xC4, Quantity: 0x16}},
		{"read holding registers", []byte{0x03, 0x00, 0x6B, 0x00, 0x03}, Request{Function: ReadHoldingRegisters, Address: 0x6B, Quantity: 3}},
		{"read input registers", []byte{0x04, 0x00, 0x08, 0x00, 0x01}, Request{Function: ReadInputRegisters, Address: 8, Quantity: 1}},
		{"write single coil", []byte{0x05, 0x00, 0xAC, 0xFF, 0x00}, Request{Function: WriteSingleCoil, Address: 0xAC, Quantity: 1, Bits: []bool{true}}},
		{"write single register", []byte{0x06, 0x00, 0x01, 0x00, 0x03}, Request{Function: WriteSingleRegister, Address: 1, Quantity: 1, Words: []uint16{3}}},
		{"write multiple coils", []byte{0x0F, 0x00, 0x13, 0x00, 0x0A, 0x02, 0xCD, 0x01},
			Request{Function: WriteMultipleCoils, Address: 0x13, Quantity: 10, Bits: bitsOf("1011001110")}},
		{"write multiple registers", []byte{0x10, 0x00, 0x01, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x01, 0x02},
			Request{Function: WriteMultipleRegisters, Address: 1, Quantity: 2, Words: []uint16{0x000A, 0x0102}}},
	}
	for _, c := range cases {
		got, exc := ParseRequest(c.pdu)
		if exc != None || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: ParseRequest = %+v, exception %d; want %+v", c.name, got, exc, c.want)
		}
		if back := got.Bytes(); !bytes.Equal(back, c.pdu) {
			t.Errorf("%s: Bytes() = % X, want the request % X", c.name, back, c.pdu)
		}
	}
}

func TestParseRequestAnswersTheExceptionOfEachFault(t *testing.T) {
	t.Parallel()
	words := func(n int) []byte { return make([]byte, 2*n) }
	cases := []struct {
		name string
		pdu  []byte
		want Exception
	}{
		{"an unknown function", []byte{0x2B, 0x0E, 0x01, 0x00}, IllegalFunction},
		{"a diagnostic function this server does not serve", []byte{0x08, 0x00, 0x00, 0x00, 0x00}, IllegalFunction},
		{"read coils, quantity 0", []byte{0x01, 0, 0, 0, 0}, IllegalDataValue},
		{"read coils, quantity 2001", []byte{0x01, 0, 0, 0x07, 0xD1}, IllegalDataValue},
		{"read discrete inputs, quantity 2001", []byte{0x02, 0, 0, 0x07, 0xD1}, IllegalDataValue},
		{"read holding registers, quantity 126", []byte{0x03, 0, 0, 0, 126}, IllegalDataValue},
		{"read input registers, quantity 0", []byte{0x04, 0, 0, 0, 0}, IllegalDataValue},
		{"read registers, a truncated request", []byte{0x03, 0, 0, 0}, IllegalDataValue},
		{"write single coil, a value that is neither FF00 nor 0000", []byte{0x05, 0, 1, 0x12, 0x34}, IllegalDataValue},
		{"write multiple coils, quantity 1969", append([]byte{0x0F, 0, 0, 0x07, 0xB1, 247}, make([]byte, 247)...), IllegalDataValue},
		{"write multiple coils, a byte count that disagrees", []byte{0x0F, 0, 0, 0, 10, 1, 0xFF}, IllegalDataValue},
		{"write multiple registers, quantity 124", append([]byte{0x10, 0, 0, 0, 124, 248}, words(124)...), IllegalDataValue},
		{"write multiple registers, quantity 0", []byte{0x10, 0, 0, 0, 0, 0}, IllegalDataValue},
		{"write multiple registers, missing data", []byte{0x10, 0, 0, 0, 2, 4, 0, 1}, IllegalDataValue},
		{"a range that runs past address 65535", []byte{0x03, 0xFF, 0xFF, 0, 2}, IllegalDataAddress},
	}
	for _, c := range cases {
		if _, exc := ParseRequest(c.pdu); exc != c.want {
			t.Errorf("%s: exception %d, want %d", c.name, exc, c.want)
		}
	}
}

func TestParseRequestAcceptsTheQuantityLimits(t *testing.T) {
	t.Parallel()
	cases := [][]byte{
		{0x01, 0, 0, 0x07, 0xD0},
		{0x02, 0, 0, 0x07, 0xD0},
		{0x03, 0, 0, 0, 125},
		{0x04, 0, 0, 0, 125},
		append([]byte{0x0F, 0, 0, 0x07, 0xB0, 246}, make([]byte, 246)...),
		append([]byte{0x10, 0, 0, 0, 123, 246}, make([]byte, 246)...),
		{0x03, 0xFF, 0xFF, 0, 1},
	}
	for _, pdu := range cases {
		if _, exc := ParseRequest(pdu); exc != None {
			t.Errorf("% X...: exception %d, want none", pdu[:5], exc)
		}
	}
}

func TestResponsesMatchTheSpecificationExamples(t *testing.T) {
	t.Parallel()
	single, _ := ParseRequest([]byte{0x05, 0x00, 0xAC, 0xFF, 0x00})
	register, _ := ParseRequest([]byte{0x06, 0x00, 0x01, 0x00, 0x03})
	coils, _ := ParseRequest([]byte{0x0F, 0x00, 0x13, 0x00, 0x0A, 0x02, 0xCD, 0x01})
	registers, _ := ParseRequest([]byte{0x10, 0x00, 0x01, 0x00, 0x02, 0x04, 0x00, 0x0A, 0x01, 0x02})
	cases := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"read coils", BitsResponse(ReadCoils, bitsOf("1011001111010110101")), []byte{0x01, 0x03, 0xCD, 0x6B, 0x05}},
		{"read discrete inputs", BitsResponse(ReadDiscreteInputs, bitsOf("0011010111011011101011")), []byte{0x02, 0x03, 0xAC, 0xDB, 0x35}},
		{"read holding registers", WordsResponse(ReadHoldingRegisters, []uint16{0x022B, 0, 0x0064}), []byte{0x03, 0x06, 0x02, 0x2B, 0, 0, 0, 0x64}},
		{"read input registers", WordsResponse(ReadInputRegisters, []uint16{0x000A}), []byte{0x04, 0x02, 0x00, 0x0A}},
		{"write single coil", EchoResponse(single), []byte{0x05, 0x00, 0xAC, 0xFF, 0x00}},
		{"write single register", EchoResponse(register), []byte{0x06, 0x00, 0x01, 0x00, 0x03}},
		{"write multiple coils", EchoResponse(coils), []byte{0x0F, 0x00, 0x13, 0x00, 0x0A}},
		{"write multiple registers", EchoResponse(registers), []byte{0x10, 0x00, 0x01, 0x00, 0x02}},
		{"exception", ExceptionResponse(ReadCoils, IllegalDataAddress), []byte{0x81, 0x02}},
	}
	for _, c := range cases {
		if !bytes.Equal(c.got, c.want) {
			t.Errorf("%s: % X, want % X", c.name, c.got, c.want)
		}
	}
}

func TestParseResponseReadsWhatTheServerEncodes(t *testing.T) {
	t.Parallel()
	read := Request{Function: ReadCoils, Address: 0x13, Quantity: 19}
	got, err := ParseResponse(read, []byte{0x01, 0x03, 0xCD, 0x6B, 0x05})
	if err != nil || !reflect.DeepEqual(got.Bits, bitsOf("1011001111010110101")) {
		t.Errorf("coils = %v, %v", got.Bits, err)
	}
	regs := Request{Function: ReadHoldingRegisters, Address: 0x6B, Quantity: 3}
	got, err = ParseResponse(regs, []byte{0x03, 0x06, 0x02, 0x2B, 0, 0, 0, 0x64})
	if err != nil || !reflect.DeepEqual(got.Words, []uint16{0x022B, 0, 0x64}) {
		t.Errorf("registers = %v, %v", got.Words, err)
	}
	got, err = ParseResponse(read, []byte{0x81, 0x02})
	if err != nil || got.Exception != IllegalDataAddress {
		t.Errorf("exception = %d, %v; want 2", got.Exception, err)
	}
	for _, bad := range [][]byte{{}, {0x03, 0x06}, {0x01, 0x02, 0xCD}, {0x81}} {
		if _, err := ParseResponse(read, bad); err == nil {
			t.Errorf("% X: no error for a response that does not answer the request", bad)
		}
	}
}
