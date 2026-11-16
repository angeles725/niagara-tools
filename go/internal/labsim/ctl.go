package labsim

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/control"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// controlTimeout bounds one exchange with a running simulator.
const controlTimeout = 5 * time.Second

// ask sends one request to the control port the command line names.
func ask(p parsed, req control.Request) (control.Response, error) {
	res, err := control.Call(p.option("control", scenario.DefaultControl), req, controlTimeout)
	if err == nil && !res.OK {
		err = errors.New(res.Error)
	}
	return res, err
}

// printValue prints the point value of a get or a set.
func printValue(p parsed, s cli.Stdio, req control.Request) (int, error) {
	res, err := ask(p, req)
	if err != nil {
		return exitFailed, err
	}
	fmt.Fprintln(s.Out, num(*res.Value))
	return exitOK, nil
}

func getCommand(p parsed, s cli.Stdio) (int, error) {
	return printValue(p, s, control.Request{Op: control.OpGet, Device: p.rest[0], Point: p.rest[1]})
}

func setCommand(p parsed, s cli.Stdio) (int, error) {
	v, err := strconv.ParseFloat(p.rest[2], 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
		return exitUsage, fmt.Errorf(msgNotNumber, p.rest[2])
	}
	return printValue(p, s, control.Request{Op: control.OpSet, Device: p.rest[0], Point: p.rest[1], Value: &v})
}

func statusCommand(p parsed, s cli.Stdio) (int, error) {
	res, err := ask(p, control.Request{Op: control.OpStatus})
	if err != nil {
		return exitFailed, err
	}
	out, _ := json.Marshal(res.Status)
	fmt.Fprintf(s.Out, "%s\n", out)
	return exitOK, nil
}

func stopCommand(p parsed, _ cli.Stdio) (int, error) {
	if _, err := ask(p, control.Request{Op: control.OpStop}); err != nil {
		return exitFailed, err
	}
	return exitOK, nil
}
