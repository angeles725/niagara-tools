package box

import (
	"strings"
	"sync"
	"testing"
)

// Field retro 2026-11-10 (client A history rollout) row 6: a dry run's read timed out once over a
// metered tunnel and the identical retry succeeded. A read is asked again once after a transport
// timeout; a write is never asked again, and a failure that is not a timeout is final.

// stallFirst answers the first request that matches with no answer until the test ends (the
// client times out), and every later request with the default answer.
func stallFirst(t *testing.T, match func(seen) bool) *scriptedStation {
	t.Helper()
	block := make(chan struct{})
	var once sync.Once
	st := newScripted(t, func(req seen) *reply {
		if !match(req) {
			return nil
		}
		stalled := false
		once.Do(func() { stalled = true })
		if stalled {
			<-block
			return &reply{Status: 500}
		}
		return nil
	})
	t.Cleanup(func() { close(block) }) // runs before the server's Close, which waits for it
	return st
}

func countKey(st *scriptedStation, key string) int {
	n := 0
	for _, k := range st.keys() {
		if k == key {
			n++
		}
	}
	return n
}

func countGets(st *scriptedStation) int {
	n := 0
	for _, r := range st.requests() {
		if r.Method == "GET" {
			n++
		}
	}
	return n
}

func TestALoadRequestThatTimedOutIsAskedAgainOnce(t *testing.T) {
	st := stallFirst(t, func(req seen) bool { return req.key() == "callssc:loadSlots" })
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	var slept []float64
	c.Sleep = func(s float64) { slept = append(slept, s) }
	_, err := c.LoadTree("station:|slot:/A", 1, LoadOptions{Attempts: 1})
	if got := countKey(st, "callssc:loadSlots"); got != 2 {
		t.Fatalf("loadSlots sent %d times, want 2 (keys %v)", got, st.keys())
	}
	if err == nil || strings.Contains(err.Error(), "transport failure") || !strings.Contains(err.Error(), "no load event") {
		t.Fatalf("err = %v, want the no-load-event answer of the retry, not the first timeout", err)
	}
	if len(slept) == 0 || slept[0] != ReadRetryDelay {
		t.Fatalf("slept %v, want the retry pause %v first", slept, ReadRetryDelay)
	}
}

func TestAnOrdReadThatTimedOutIsAskedAgainOnce(t *testing.T) {
	st := stallFirst(t, func(req seen) bool { return req.Method == "GET" })
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	_, _, err := c.GetOrd("station:|slot:/", 0, 0)
	if got := countGets(st); got != 2 {
		t.Fatalf("GET sent %d times, want 2", got)
	}
	if err == nil || strings.Contains(err.Error(), "timed out") { // the retry got the default 404
		t.Fatalf("err = %v, want the retry's HTTP 404, not the first timeout", err)
	}
}

func TestAReadThatTimedOutTwiceIsAFailureAfterTwoRequests(t *testing.T) {
	block := make(chan struct{})
	st := newScripted(t, func(req seen) *reply {
		if req.Method == "GET" {
			<-block
		}
		return nil
	})
	t.Cleanup(func() { close(block) })
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	_, _, err := c.GetOrd("station:|slot:/", 0, 0)
	if e := boxErr(t, err, KindBox); !strings.Contains(e.Msg, "timed out") {
		t.Fatalf("msg = %q", e.Msg)
	}
	if got := countGets(st); got != 2 {
		t.Fatalf("GET sent %d times, want exactly 2", got)
	}
}

func TestAWriteThatTimedOutIsNeverAskedAgain(t *testing.T) {
	st := stallFirst(t, func(req seen) bool { return req.key() == "callssc:syncTo" })
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	_, err := c.SSC("syncTo", obj())
	if e := boxErr(t, err, KindBox); e.Msg != "transport failure: timed out" {
		t.Fatalf("msg = %q", e.Msg)
	}
	if got := countKey(st, "callssc:syncTo"); got != 1 {
		t.Fatalf("syncTo sent %d times, want 1", got)
	}
}

func TestAReadThatFailedWithoutATimeoutIsNotAskedAgain(t *testing.T) {
	st := newScripted(t, func(req seen) *reply {
		if req.Method == "GET" {
			return &reply{Status: 500}
		}
		return nil
	})
	c := newClient(t, st.URL, nil)
	_, _, err := c.GetOrd("station:|slot:/", 0, 0)
	if e := boxErr(t, err, KindBox); e.Msg != "HTTP 500 from station" {
		t.Fatalf("msg = %q", e.Msg)
	}
	if got := countGets(st); got != 1 {
		t.Fatalf("%d GETs, want 1", got)
	}
}
