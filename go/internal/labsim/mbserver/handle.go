package mbserver

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/modbus"
	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// tables says which table each function code addresses.
var tables = map[byte]regmap.Table{
	modbus.ReadCoils:              regmap.Coils,
	modbus.ReadDiscreteInputs:     regmap.DiscreteInputs,
	modbus.ReadHoldingRegisters:   regmap.HoldingRegisters,
	modbus.ReadInputRegisters:     regmap.InputRegisters,
	modbus.WriteSingleCoil:        regmap.Coils,
	modbus.WriteSingleRegister:    regmap.HoldingRegisters,
	modbus.WriteMultipleCoils:     regmap.Coils,
	modbus.WriteMultipleRegisters: regmap.HoldingRegisters,
}

// reasons names each exception in the request log.
var reasons = map[modbus.Exception]string{
	modbus.IllegalFunction:     reqlog.ReasonIllegalFunction,
	modbus.IllegalDataAddress:  reqlog.ReasonIllegalDataAddress,
	modbus.IllegalDataValue:    reqlog.ReasonIllegalDataValue,
	modbus.ServerDeviceFailure: reqlog.ReasonDeviceFailure,
	modbus.GatewayTargetFailed: reqlog.ReasonUnknownUnit,
}

// answer is what a request gets: a response PDU (nil for none), how long it is
// held, and its log entry.
type answer struct {
	pdu   []byte
	delay time.Duration
	entry reqlog.Entry
}

func (a *answer) exception(function byte, e modbus.Exception) answer {
	a.pdu = modbus.ExceptionResponse(function, e)
	a.entry.Result, a.entry.Exception, a.entry.Reason = reqlog.ResultException, int(e), reasons[e]
	return a.held()
}

func (a *answer) silent(reason string) answer {
	a.entry.Result, a.entry.Reason = reqlog.ResultNoResponse, reason
	return *a
}

func (a *answer) held() answer {
	a.entry.DelayMS = a.delay.Milliseconds()
	return *a
}

// handle answers one Modbus frame, in the order a device decides: is the unit
// there, is it online, is the request well formed, do its addresses exist (and
// accept a write), has the device failed; only then is it executed.
func (s *Server) handle(f modbus.Frame, remote string) answer {
	elapsed := s.Elapsed()
	function := f.PDU[0]
	a := &answer{entry: reqlog.Entry{
		Time:      s.opts.Clock().UTC().Format(time.RFC3339Nano),
		ElapsedMS: elapsed.Milliseconds(),
		Remote:    remote, Unit: int(f.Unit), Function: int(function),
	}}
	req, exc := modbus.ParseRequest(f.PDU)
	a.entry.Address, a.entry.Quantity = int(req.Address), int(req.Quantity)
	a.entry.Written = append(ints(req.Words), flags(req.Bits)...)
	unit, ok := s.opts.Units[f.Unit]
	if !ok {
		if s.opts.GatewayException {
			return a.exception(function, modbus.GatewayTargetFailed)
		}
		return a.silent(reqlog.ReasonUnknownUnit)
	}
	device := unit.Device()
	a.entry.Device = device.Name()
	if device.Offline(elapsed) {
		return a.silent(reqlog.ReasonOffline)
	}
	a.delay = device.Delay(nil)
	if exc != modbus.None {
		return a.exception(function, exc)
	}
	table, write := tables[function], modbus.IsWrite(function)
	points, ok := unit.Resolve(table, req.Address, int(req.Quantity), write)
	if !ok {
		return a.exception(function, modbus.IllegalDataAddress)
	}
	a.delay = device.Delay(points)
	if !device.Admit(points, !write) {
		return a.exception(function, modbus.ServerDeviceFailure)
	}
	a.entry.Result = reqlog.ResultOK
	switch {
	case write && table.Bits():
		unit.WriteBits(req.Address, req.Bits, elapsed)
		a.pdu = modbus.EchoResponse(req)
	case write:
		unit.WriteWords(req.Address, req.Words, elapsed)
		a.pdu = modbus.EchoResponse(req)
	case table.Bits():
		bits := unit.ReadBits(table, req.Address, int(req.Quantity), elapsed)
		a.pdu, a.entry.Read = modbus.BitsResponse(function, bits), flags(bits)
	default:
		words := unit.ReadWords(table, req.Address, int(req.Quantity), elapsed)
		a.pdu, a.entry.Read = modbus.WordsResponse(function, words), ints(words)
	}
	return a.held()
}

func ints(words []uint16) []int {
	var out []int
	for _, w := range words {
		out = append(out, int(w))
	}
	return out
}

func flags(bits []bool) []int {
	var out []int
	for _, on := range bits {
		if on {
			out = append(out, 1)
		} else {
			out = append(out, 0)
		}
	}
	return out
}
