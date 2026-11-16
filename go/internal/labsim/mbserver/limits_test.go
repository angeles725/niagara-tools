package mbserver

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

func TestEveryRequestIsLoggedWithWhatWasAskedAndAnswered(t *testing.T) {
	t.Parallel()
	s := start(t, nil)
	c := s.dial()
	s.advance(1500 * time.Millisecond)
	ask(t, c, 1, 0x10, 0, 1, 0, 2, 4, 0x40, 0x40, 0, 0)
	ask(t, c, 1, 0x01, 0, 0, 0, 3)
	ask(t, c, 1, 0x03, 0, 3, 0, 1)
	ask(t, c, 2, 0x04, 0, 0, 0, 1)
	remote := c.LocalAddr().String()
	stamp := "2026-01-02T03:04:06.5Z"
	want := []reqlog.Entry{
		{Time: stamp, ElapsedMS: 1500, Remote: remote, Unit: 1, Device: "unit-a", Function: 16, Address: 1, Quantity: 2,
			Written: []int{0x4040, 0}, Result: reqlog.ResultOK},
		{Time: stamp, ElapsedMS: 1500, Remote: remote, Unit: 1, Device: "unit-a", Function: 1, Address: 0, Quantity: 3,
			Read: []int{1, 0, 1}, Result: reqlog.ResultOK},
		{Time: stamp, ElapsedMS: 1500, Remote: remote, Unit: 1, Device: "unit-a", Function: 3, Address: 3, Quantity: 1,
			Result: reqlog.ResultException, Exception: 2, Reason: reqlog.ReasonIllegalDataAddress},
		{Time: stamp, ElapsedMS: 1500, Remote: remote, Unit: 2, Device: "unit-b", Function: 4, Address: 0, Quantity: 1,
			Read: []int{5}, Result: reqlog.ResultOK, DelayMS: 150},
	}
	if got := s.logged(); !reflect.DeepEqual(got, want) {
		t.Errorf("log =\n%+v\nwant\n%+v", got, want)
	}
	if got := s.server.Stats(); got.Requests != 4 || got.Connections != 1 {
		t.Errorf("Stats = %+v, want 4 requests on 1 connection", got)
	}
}

func TestAConnectionOverTheLimitIsClosedAtOnce(t *testing.T) {
	t.Parallel()
	s := start(t, func(o *Options) { o.MaxConnections = 1 })
	first := s.dial()
	ask(t, first, 1, 0x04, 0, 0, 0, 1)
	second := s.dial()
	if !closedByServer(second) {
		t.Errorf("second connection: the server did not close the connection")
	}
	ask(t, first, 1, 0x04, 0, 0, 0, 1)
	if got := s.server.Stats().Refused; got != 1 {
		t.Errorf("Refused = %d, want 1", got)
	}
}

func TestAClientThatStopsMidFrameIsDropped(t *testing.T) {
	t.Parallel()
	s := start(t, func(o *Options) { o.IdleTimeout = 30 * time.Millisecond })
	c := s.dial()
	if _, err := c.Write([]byte{0, 1, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if !closedByServer(c) {
		t.Errorf("stuck client: the server did not close the connection")
	}
}

func TestCancellingTheContextStopsTheServerAndItsConnections(t *testing.T) {
	t.Parallel()
	s := start(t, nil)
	c := s.dial()
	ask(t, c, 1, 0x04, 0, 0, 0, 1)
	s.stop()
	select {
	case err := <-s.done:
		if err != nil {
			t.Errorf("Serve = %v, want nil after a stop", err)
		}
		s.done <- nil
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
	}
	if !closedByServer(c) {
		t.Errorf("open connection after the stop: the server did not close the connection")
	}
}

func TestTheRealSleepEndsWithItsTimerOrItsContext(t *testing.T) {
	t.Parallel()
	if !realSleep(context.Background(), time.Microsecond) {
		t.Error("a sleep that ran its time reported an interruption")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if realSleep(ctx, time.Hour) {
		t.Error("a sleep under an ended context reported that it ran its time")
	}
}
