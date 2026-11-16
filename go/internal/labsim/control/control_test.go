package control

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

type bench struct {
	addr    string
	stopped chan struct{}
	done    chan error
}

func start(t *testing.T) *bench {
	t.Helper()
	dev := model.NewDevice(model.DeviceSpec{Name: "unit-a", Points: []model.PointSpec{
		{Name: "setpoint", Initial: 21.5, Writable: true},
		{Name: "ramp", Generator: model.Ramp(0, 100, time.Second)},
	}})
	b := &bench{stopped: make(chan struct{}, 1), done: make(chan error, 1)}
	srv := &Server{
		Devices: []*model.Device{dev},
		Elapsed: func() time.Duration { return 250 * time.Millisecond },
		Status:  func() Status { return Status{Scenario: "bench", Requests: 7} },
		Stop:    func() { b.stopped <- struct{}{} },
	}
	ln, err := Listen("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	b.addr = ln.Addr().String()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { b.done <- srv.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-b.done:
		case <-time.After(5 * time.Second):
			t.Error("the control server did not stop")
		}
	})
	return b
}

func value(v float64) *float64 { return &v }

func call(t *testing.T, addr string, req Request) Response {
	t.Helper()
	res, err := Call(addr, req, 5*time.Second)
	if err != nil {
		t.Fatalf("Call(%+v): %v", req, err)
	}
	return res
}

func TestGetReadsAPointAtTheSimulatorTime(t *testing.T) {
	t.Parallel()
	b := start(t)
	if res := call(t, b.addr, Request{Op: OpGet, Device: "unit-a", Point: "setpoint"}); !res.OK || res.Value == nil || *res.Value != 21.5 {
		t.Errorf("setpoint: %+v", res)
	}
	if res := call(t, b.addr, Request{Op: OpGet, Device: "unit-a", Point: "ramp"}); !res.OK || *res.Value != 25 {
		t.Errorf("ramp at 250 ms: %+v, want 25", res)
	}
}

func TestSetHoldsAPointEvenAGeneratedOne(t *testing.T) {
	t.Parallel()
	b := start(t)
	if res := call(t, b.addr, Request{Op: OpSet, Device: "unit-a", Point: "ramp", Value: value(42)}); !res.OK || *res.Value != 42 {
		t.Errorf("set: %+v", res)
	}
	if res := call(t, b.addr, Request{Op: OpGet, Device: "unit-a", Point: "ramp"}); *res.Value != 42 {
		t.Errorf("get after set: %+v, want 42", res)
	}
}

func TestStatusAndStopReachTheirHooks(t *testing.T) {
	t.Parallel()
	b := start(t)
	res := call(t, b.addr, Request{Op: OpStatus})
	if !res.OK || res.Status == nil || res.Status.Scenario != "bench" || res.Status.Requests != 7 {
		t.Errorf("status: %+v", res)
	}
	if res := call(t, b.addr, Request{Op: OpStop}); !res.OK {
		t.Errorf("stop: %+v", res)
	}
	select {
	case <-b.stopped:
	case <-time.After(5 * time.Second):
		t.Error("stop did not reach the simulator")
	}
}

func TestARequestThatCannotBeServedSaysWhy(t *testing.T) {
	t.Parallel()
	b := start(t)
	for want, req := range map[string]Request{
		`no device "unit-z" (devices: unit-a)`:                      {Op: OpGet, Device: "unit-z", Point: "setpoint"},
		`device "unit-a" has no point "x" (points: setpoint, ramp)`: {Op: OpSet, Device: "unit-a", Point: "x", Value: value(1)},
		`set needs a value`:  {Op: OpSet, Device: "unit-a", Point: "setpoint"},
		`unknown op "reset"`: {Op: "reset"},
	} {
		if res := call(t, b.addr, req); res.OK || res.Error != want {
			t.Errorf("%+v: %+v, want the error %q", req, res, want)
		}
	}
}

func TestALineThatIsNotARequestIsRefused(t *testing.T) {
	t.Parallel()
	b := start(t)
	for _, line := range []string{"POST / HTTP/1.1", `{"op":"get","extra":1}`, strings.Repeat("x", 2*maxLine)} {
		c, err := net.Dial("tcp", b.addr)
		if err != nil {
			t.Fatal(err)
		}
		c.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err := c.Write([]byte(line + "\n")); err != nil {
			t.Fatal(err)
		}
		reply, _ := bufio.NewReader(c).ReadString('\n')
		c.Close()
		// A line over the limit is closed with bytes unread, which may reset the
		// connection before the refusal arrives; it must never be served.
		if reply == "" && len(line) > maxLine {
			continue
		}
		if !strings.HasPrefix(reply, `{"ok":false,"error":"not a control request`) {
			t.Errorf("%.20q: reply %q", line, reply)
		}
	}
}

func TestTheControlPortNeverLeavesThisMachine(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"0.0.0.0:0", ":0", "192.0.2.10:1503"} {
		if ln, err := Listen(addr); err == nil {
			ln.Close()
			t.Errorf("Listen(%q) accepted a non-loopback address", addr)
		}
		if _, err := Call(addr, Request{Op: OpStatus}, time.Second); err == nil || !strings.Contains(err.Error(), "loopback") {
			t.Errorf("Call(%q) = %v, want a loopback refusal before any dial", addr, err)
		}
	}
}

func TestCallReportsASimulatorThatIsNotRunning(t *testing.T) {
	t.Parallel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	if _, err := Call(addr, Request{Op: OpStatus}, time.Second); err == nil {
		t.Error("Call to a closed port returned no error")
	}
}
