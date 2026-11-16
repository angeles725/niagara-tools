package labsim

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/control"
	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
	"github.com/angeles725/niagara-tools/go/internal/labsim/mbserver"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// frontEnd is a field protocol served over the scenario's devices. Modbus TCP
// is the only one today; another protocol adds its server here and its fields
// to the scenario, and shares the devices, the control port and the stop.
type frontEnd struct {
	listener net.Listener
	serve    func(ctx context.Context, ln net.Listener) error
}

// listenAddress decides where the field protocol listens: the scenario's
// address when it is loopback, or whatever --listen names. A scenario alone
// never binds outside this machine.
func listenAddress(f *scenario.File, flag string, given bool) (string, error) {
	if given {
		if _, _, err := net.SplitHostPort(flag); err != nil {
			return "", fmt.Errorf(msgBadListen, flag)
		}
		return flag, nil
	}
	if err := loopback.Check(f.Listen); err != nil {
		return "", fmt.Errorf(msgExposed, err, f.Listen)
	}
	return f.Listen, nil
}

// runCommand serves a scenario until a signal or a control stop.
func runCommand(p parsed, s cli.Stdio) (int, error) {
	f, ok := load(p.rest[0], s)
	if !ok {
		return exitFailed, nil
	}
	flag, given := p.options["listen"]
	listen, err := listenAddress(f, flag, given)
	if err != nil {
		return exitFailed, err
	}
	logTo, closeLog, err := openLog(p.option("log", ""), s.Out)
	if err != nil {
		return exitFailed, err
	}
	defer closeLog()
	ctl, err := control.Listen(p.option("control", f.Control))
	if err != nil {
		return exitFailed, err
	}
	defer ctl.Close()
	bus, err := net.Listen("tcp", listen)
	if err != nil {
		return exitFailed, err
	}
	defer bus.Close()

	built := f.Build()
	server := mbserver.New(mbserver.Options{
		Units:            built.Units,
		GatewayException: f.UnknownUnit == scenario.UnknownGateway,
		MaxConnections:   f.MaxConnections,
		IdleTimeout:      time.Duration(f.IdleTimeoutMS) * time.Millisecond,
		Log:              logTo.Log,
	})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	panel := &control.Server{
		Devices: built.Devices,
		Elapsed: server.Elapsed,
		Status:  func() control.Status { return status(f, built, server, bus.Addr().String(), ctl.Addr().String()) },
		Stop:    stop,
	}
	ends := []frontEnd{{bus, server.Serve}, {ctl, panel.Serve}}
	ready, _ := json.Marshal(map[string]string{
		"event": "listening", "scenario": f.Name, "listen": bus.Addr().String(), "control": ctl.Addr().String(),
	})
	fmt.Fprintf(s.Out, "%s\n", ready)

	failures := make(chan error, len(ends))
	for _, end := range ends {
		go func() {
			err := end.serve(ctx, end.listener)
			stop() // one front end down stops the others
			failures <- err
		}()
	}
	for range ends {
		if e := <-failures; e != nil && err == nil {
			err = e
		}
	}
	if err != nil {
		return exitFailed, err
	}
	return exitOK, nil
}

// openLog opens the request log: none, standard output for "-", or a file
// that is appended to.
func openLog(path string, stdout io.Writer) (*reqlog.Writer, func(), error) {
	switch path {
	case "":
		return nil, func() {}, nil
	case "-":
		return reqlog.NewWriter(stdout), func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}
	return reqlog.NewWriter(file), func() { file.Close() }, nil
}

// status describes the running simulator for the control port.
func status(f *scenario.File, built scenario.Built, server *mbserver.Server, listen, ctl string) control.Status {
	stats, elapsed := server.Stats(), server.Elapsed()
	st := control.Status{
		Scenario: f.Name, ElapsedMS: elapsed.Milliseconds(), Listen: listen, Control: ctl,
		Requests: stats.Requests, Connections: stats.Connections, Refused: stats.Refused,
	}
	for i, dev := range built.Devices {
		st.Devices = append(st.Devices, control.DeviceStatus{
			Name: dev.Name(), Unit: f.Devices[i].Unit, Points: len(dev.Points()),
			Reads: dev.Reads(), Offline: dev.Offline(elapsed),
		})
	}
	return st
}
