// Package modbus is the Modbus TCP wire codec of lab-sim: the MBAP frame, the
// request and response PDUs of function codes 1-6, 15 and 16, and the exception
// responses. It holds no state and knows nothing of a device: the caller maps a
// parsed Request onto its registers.
package modbus

// The function codes this codec reads and writes.
const (
	ReadCoils              byte = 0x01
	ReadDiscreteInputs     byte = 0x02
	ReadHoldingRegisters   byte = 0x03
	ReadInputRegisters     byte = 0x04
	WriteSingleCoil        byte = 0x05
	WriteSingleRegister    byte = 0x06
	WriteMultipleCoils     byte = 0x0F
	WriteMultipleRegisters byte = 0x10
)

// Exception is a Modbus exception code; None means the request is well formed.
type Exception byte

// The exception codes a simulated device answers.
const (
	None                Exception = 0x00
	IllegalFunction     Exception = 0x01
	IllegalDataAddress  Exception = 0x02
	IllegalDataValue    Exception = 0x03
	ServerDeviceFailure Exception = 0x04
	GatewayTargetFailed Exception = 0x0B
)

// exceptionFlag is set on the function code of an exception response.
const exceptionFlag byte = 0x80

// The quantity limits of the Modbus Application Protocol V1.1b3, section 6.
const (
	MaxReadBits   = 2000 // function codes 1 and 2
	MaxReadWords  = 125  // function codes 3 and 4
	MaxWriteBits  = 1968 // function code 15
	MaxWriteWords = 123  // function code 16
)

// The sizes of a Modbus TCP frame.
const (
	MaxPDU     = 253 // function code plus data
	headerSize = 7   // transaction, protocol, length, unit
	addresses  = 1 << 16
)

// The two values a write-single-coil request may carry.
const (
	coilOn  uint16 = 0xFF00
	coilOff uint16 = 0x0000
)

// shape describes how a function code's request is laid out.
type shape struct {
	maxQuantity int
	bits        bool // the data are coils, not registers
	single      bool // function codes 5 and 6: one value, no quantity field
	multiple    bool // function codes 15 and 16: quantity, byte count, data
}

// shapes is the table of supported function codes; any other one is answered
// with IllegalFunction.
var shapes = map[byte]shape{
	ReadCoils:              {maxQuantity: MaxReadBits, bits: true},
	ReadDiscreteInputs:     {maxQuantity: MaxReadBits, bits: true},
	ReadHoldingRegisters:   {maxQuantity: MaxReadWords},
	ReadInputRegisters:     {maxQuantity: MaxReadWords},
	WriteSingleCoil:        {maxQuantity: 1, bits: true, single: true},
	WriteSingleRegister:    {maxQuantity: 1, single: true},
	WriteMultipleCoils:     {maxQuantity: MaxWriteBits, bits: true, multiple: true},
	WriteMultipleRegisters: {maxQuantity: MaxWriteWords, multiple: true},
}

// IsWrite reports whether function changes the device (5, 6, 15, 16).
func IsWrite(function byte) bool {
	s, ok := shapes[function]
	return ok && (s.single || s.multiple)
}
