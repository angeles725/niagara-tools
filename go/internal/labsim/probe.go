package labsim

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
	"github.com/angeles725/niagara-tools/go/internal/labsim/modbus"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// probeResult is the one JSON line a probe prints.
type probeResult struct {
	Unit      int   `json:"unit"`
	Function  int   `json:"function"`
	Address   int   `json:"address"`
	Quantity  int   `json:"quantity"`
	Values    []int `json:"values,omitempty"`
	Exception int   `json:"exception,omitempty"`
}

// probeCommand sends one Modbus request to a simulator on this machine. It is
// a way to see the simulator answer before a driver is pointed at it, not a
// field tool: an address that is not loopback is refused.
func probeCommand(p parsed, s cli.Stdio) (int, error) {
	unit, err := whole(p, "unit", "1", 0, 255)
	if err != nil {
		return exitUsage, err
	}
	timeout, err := whole(p, "timeout-ms", "3000", 1, 600000)
	if err != nil {
		return exitUsage, err
	}
	req, err := probeRequest(p)
	if err != nil {
		return exitUsage, err
	}
	addr := p.option("addr", scenario.DefaultListen)
	if err := loopback.Check(addr); err != nil {
		return exitFailed, fmt.Errorf("probe address %v", err)
	}
	res, err := exchange(addr, byte(unit), req, time.Duration(timeout)*time.Millisecond)
	if err != nil {
		return exitFailed, fmt.Errorf(msgProbeAnswer, addr, timeout, err)
	}
	out := probeResult{Unit: unit, Function: int(req.Function), Address: int(req.Address), Quantity: int(req.Quantity),
		Exception: int(res.Exception)}
	for _, w := range res.Words {
		out.Values = append(out.Values, int(w))
	}
	for _, on := range res.Bits {
		out.Values = append(out.Values, bit(on))
	}
	line, _ := json.Marshal(out)
	fmt.Fprintf(s.Out, "%s\n", line)
	if res.Exception != modbus.None {
		return exitFailed, nil
	}
	return exitOK, nil
}

func bit(on bool) int {
	if on {
		return 1
	}
	return 0
}

// whole reads an option as a whole number within limits.
func whole(p parsed, name, fallback string, low, high int) (int, error) {
	text := p.option(name, fallback)
	n, err := strconv.Atoi(text)
	if err != nil || n < low || n > high {
		return 0, fmt.Errorf(msgProbeNumber, name, text, low, high)
	}
	return n, nil
}

// probeRequest builds the request the options describe and refuses one a
// device would answer with an exception for its shape alone.
func probeRequest(p parsed) (modbus.Request, error) {
	function, err := strconv.Atoi(p.option("fn", ""))
	if err != nil || function < 1 || function > 255 || !isProbeFunction(byte(function)) {
		return modbus.Request{}, fmt.Errorf(msgProbeFunction)
	}
	address, err := whole(p, "address", "", 0, scenario.MaxAddress)
	if err != nil {
		return modbus.Request{}, err
	}
	req := modbus.Request{Function: byte(function), Address: uint16(address)}
	_, counted := p.options["count"]
	list, valued := p.options["values"]
	if !modbus.IsWrite(req.Function) {
		if !counted || valued {
			return req, fmt.Errorf(msgProbeCount, function)
		}
		count, err := whole(p, "count", "", 1, modbus.MaxReadBits)
		req.Quantity = uint16(count)
		return checked(req, err)
	}
	if !valued || counted {
		return req, fmt.Errorf(msgProbeValues, function)
	}
	coils := function == int(modbus.WriteSingleCoil) || function == int(modbus.WriteMultipleCoils)
	for _, text := range strings.Split(list, ",") {
		limit := 65535
		if coils {
			limit = 1
		}
		v, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || v < 0 || v > limit {
			return req, fmt.Errorf(msgProbeNumber, "values", text, 0, limit)
		}
		if coils {
			req.Bits = append(req.Bits, v == 1)
		} else {
			req.Words = append(req.Words, uint16(v))
		}
	}
	req.Quantity = uint16(len(req.Bits) + len(req.Words))
	single := function == int(modbus.WriteSingleCoil) || function == int(modbus.WriteSingleRegister)
	if single && req.Quantity != 1 {
		return req, fmt.Errorf(msgProbeOne, function)
	}
	return checked(req, nil)
}

// isProbeFunction reports whether the codec can build a request of function.
func isProbeFunction(function byte) bool {
	_, exc := modbus.ParseRequest([]byte{function})
	return exc != modbus.IllegalFunction
}

// checked passes the request through the server's own parser.
func checked(req modbus.Request, err error) (modbus.Request, error) {
	if err != nil {
		return req, err
	}
	if _, exc := modbus.ParseRequest(req.Bytes()); exc != modbus.None {
		return req, fmt.Errorf(msgProbeRequest, exc)
	}
	return req, nil
}

// exchange sends one request and reads its response.
func exchange(addr string, unit byte, req modbus.Request, timeout time.Duration) (modbus.Response, error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return modbus.Response{}, err
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
		return modbus.Response{}, err
	}
	if _, err := c.Write(modbus.Frame{Transaction: 1, Unit: unit, PDU: req.Bytes()}.Bytes()); err != nil {
		return modbus.Response{}, err
	}
	frame, err := modbus.ReadFrame(c)
	if err != nil {
		return modbus.Response{}, err
	}
	return modbus.ParseResponse(req, frame.PDU)
}
