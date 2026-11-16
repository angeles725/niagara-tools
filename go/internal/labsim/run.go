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

// The protocols of the front ends, as status names them.
const (
	protocolModbus  = "modbus-tcp"
	protocolBACnet  = "bacnet-ip"
	protocolControl = "control"
)

// frontEnd is one listening address served over the scenario's devices: the
// Modbus TCP listener, one BACnet/IP socket per BACnet device, the control
// port. They share the devices, the clock and the stop.
type frontEnd struct {
	control.FrontEnd
	serve func(ctx context.Context) error
	close func()
}

// listenAddress decides where Modbus listens: the scenario's address when it
// is loopback, or whatever --listen names. A scenario alone never binds
// outside this machine.
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
	broadcast, err := broadcastAddress(p)
	if err != nil {
		return exitUsage, err
	}
	flag, given := p.options["listen"]
	listen, err := listenAddress(f, flag, given)
	if err != nil {
		return exitFailed, err
	}
	flag, given = p.options["bacnet-listen"]
	bacnetListen, err := bacnetAddresses(f, flag, given)
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
	ends := []frontEnd{{FrontEnd: control.FrontEnd{Protocol: protocolModbus, Address: bus.Addr().String()},
		serve: func(ctx context.Context) error { return server.Serve(ctx, bus) }}}
	field, err := bacnetEnds(built, bacnetListen, broadcast, server.Elapsed, logTo)
	for _, end := range field.ends {
		defer end.close()
	}
	if err != nil {
		return exitFailed, err
	}
	ends = append(ends, field.ends...)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	panel := &control.Server{Devices: built.Devices, Elapsed: server.Elapsed, Stop: stop, Changed: field.changed}
	ends = append(ends, frontEnd{FrontEnd: control.FrontEnd{Protocol: protocolControl, Address: ctl.Addr().String()},
		serve: func(ctx context.Context) error { return panel.Serve(ctx, ctl) }})
	panel.Status = func() control.Status { return status(f, built, server, ends) }
	fmt.Fprintf(s.Out, "%s\n", readyLine(f, ends))

	failures := make(chan error, len(ends))
	for _, end := range ends {
		go func() {
			err := end.serve(ctx)
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

// readyLine is the one line run prints when every front end listens, with the
// addresses really bound (useful with port 0).
func readyLine(f *scenario.File, ends []frontEnd) []byte {
	ready := map[string]any{"event": "listening", "scenario": f.Name}
	var field []map[string]string
	for _, end := range ends {
		switch end.Protocol {
		case protocolModbus:
			ready["listen"] = end.Address
		case protocolControl:
			ready["control"] = end.Address
		case protocolBACnet:
			field = append(field, map[string]string{"device": end.Device, "address": end.Address})
		}
	}
	if field != nil {
		ready["bacnet"] = field
	}
	line, _ := json.Marshal(ready)
	return line
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
func status(f *scenario.File, built scenario.Built, server *mbserver.Server, ends []frontEnd) control.Status {
	stats, elapsed := server.Stats(), server.Elapsed()
	st := control.Status{
		Scenario: f.Name, ElapsedMS: elapsed.Milliseconds(),
		Requests: stats.Requests, Connections: stats.Connections, Refused: stats.Refused,
	}
	for _, end := range ends {
		st.FrontEnds = append(st.FrontEnds, end.FrontEnd)
		switch end.Protocol {
		case protocolModbus:
			st.Listen = end.Address
		case protocolControl:
			st.Control = end.Address
		}
	}
	for i, dev := range built.Devices {
		st.Devices = append(st.Devices, control.DeviceStatus{
			Name: dev.Name(), Unit: f.Devices[i].Unit, Points: len(dev.Points()),
			Reads: dev.Reads(), Offline: dev.Offline(elapsed),
		})
	}
	return st
}
