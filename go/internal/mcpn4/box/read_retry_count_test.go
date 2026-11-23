package box

import (
	"context"
	"testing"
)

const loadSlotsKey = "callssc:loadSlots"

func isLoadSlots(req seen) bool { return req.key() == loadSlotsKey }

// A timeout does not prove the station dropped the request: it may answer the first loadSlots as well as
// the second, so the second is counted too and a stray second load event is never trusted as "the one".
func TestARetriedLoadRequestCountsBothAsOutstanding(t *testing.T) {
	st := stallFirst(t, isLoadSlots)
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	_, err := c.LoadTree("station:|slot:/A", 1, LoadOptions{Attempts: 1})
	if got := countKey(st, loadSlotsKey); got != 2 || err == nil {
		t.Fatalf("loadSlots sent %d times, err %v; want 2 and the no-load-event answer", got, err)
	}
	if got := c.LoadsOutstanding(); got != 2 {
		t.Fatalf("outstanding = %d after a timed-out request and its unanswered retry, want 2", got)
	}
}

// A request that failed outright (no timeout) was rejected, not lost: nothing stays outstanding.
func TestARejectedLoadRequestIsNotOutstanding(t *testing.T) {
	st := newScripted(t, func(req seen) *reply {
		if isLoadSlots(req) {
			return &reply{Status: 500}
		}
		return nil
	})
	c := newClient(t, st.URL, nil)
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LoadTree("station:|slot:/A", 1, LoadOptions{Attempts: 1}); err == nil {
		t.Fatal("want the HTTP 500")
	}
	if got := countKey(st, loadSlotsKey); got != 1 || c.LoadsOutstanding() != 0 {
		t.Fatalf("loadSlots sent %d times, outstanding %d; want 1 and 0", got, c.LoadsOutstanding())
	}
}

// An interrupted client does not ask again, whatever failed: the pause is skipped and nothing is re-sent.
func TestAnInterruptedClientDoesNotRetryATimedOutRead(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	st := stallFirst(t, isLoadSlots)
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3; c.Interrupt = ctx })
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	slept := false
	c.Sleep = func(float64) { slept = true }
	cancel()
	if c.retryRead(&transportFailure{"timed out"}) || slept {
		t.Fatalf("an interrupted client asked a timed-out read again (slept %v)", slept)
	}
	if _, err := c.LoadTree("station:|slot:/A", 1, LoadOptions{Attempts: 1}); err == nil {
		t.Fatal("want a failure")
	}
	if got := countKey(st, loadSlotsKey); got > 1 || slept {
		t.Fatalf("loadSlots sent %d times (slept %v), want at most 1 (keys %v)", got, slept, st.keys())
	}
}
