package control

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
)

// Call sends one request to the simulator's control port at addr and returns
// its response. An address that is not loopback is refused before any dial. A
// response with OK false is not an error here: the caller reads its Error.
func Call(addr string, req Request, timeout time.Duration) (Response, error) {
	if err := loopback.Check(addr); err != nil {
		return Response{}, fmt.Errorf("control address %v", err)
	}
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return Response{}, fmt.Errorf("no simulator on control port %s: %w", addr, err)
	}
	defer c.Close()
	if err := c.SetDeadline(time.Now().Add(timeout)); err != nil {
		return Response{}, err
	}
	out, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}
	if _, err := c.Write(append(out, '\n')); err != nil {
		return Response{}, err
	}
	line, err := bufio.NewReader(c).ReadBytes('\n')
	if err != nil {
		return Response{}, fmt.Errorf("control port %s closed without a response: %w", addr, err)
	}
	var res Response
	if err := json.Unmarshal(line, &res); err != nil {
		return Response{}, fmt.Errorf("control port %s answered something else than a response: %w", addr, err)
	}
	return res, nil
}
