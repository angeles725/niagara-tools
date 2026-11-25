package box

import (
	"reflect"
	"testing"
)

// More than MaxFrameMessages ords go in batches, one loadSlots frame per poll cycle: a retry of
// a batch never re-sends an earlier one, and loadsOutstanding counts exactly what is in doubt.

func loadSeq(ls *loadStation) []string {
	var out []string
	for _, k := range ls.keys() {
		if k == "pollchgs" || k == "callssc:loadSlots" {
			out = append(out, k)
		}
	}
	return out
}

func TestLoadManyOf120OrdsIsThreeSequentialPollCycles(t *testing.T) {
	ls := newLoadStation(t)
	c := openClient(t, ls)
	ords := ordsOf(120)
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords, rs)
	const p, l = "pollchgs", "callssc:loadSlots"
	if want := []string{p, l, p, l, p, l, p}; !reflect.DeepEqual(loadSeq(ls), want) {
		t.Errorf("requests %v, want %v", loadSeq(ls), want)
	}
	if want := [][]string{ords[:50], ords[50:100], ords[100:]}; !reflect.DeepEqual(ls.frameOrds, want) {
		t.Errorf("loadSlots frames carried %d, %d, %d ords, want 50, 50, 20 in order",
			len(ls.frameOrds[0]), len(ls.frameOrds[1]), len(ls.frameOrds[2]))
	}
	if c.LoadsOutstanding() != 0 {
		t.Errorf("%d loads outstanding", c.LoadsOutstanding())
	}
}

func TestLoadManyTimeoutOnTheSecondBatchRetriesOnlyThatBatch(t *testing.T) {
	ls := newLoadStation(t)
	ls.stallLoadFrame(t, 2)
	c := newClient(t, ls.URL, func(c *Config) { c.Timeout = 0.3 })
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	c.Sleep = func(float64) {}
	ords := ordsOf(70)
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords, rs)
	if want := [][]string{ords[:50], ords[50:], ords[50:]}; !reflect.DeepEqual(ls.frameOrds, want) {
		t.Errorf("frames of %d ords, want 50, 20 (timed out), 20 (retry): the first batch is never re-sent",
			len(ls.frameOrds))
	}
	// the timed-out request may still have been accepted: its 20 loads stay in doubt, the first
	// batch's 50 are settled
	if c.LoadsOutstanding() != 20 {
		t.Errorf("%d loads outstanding, want 20", c.LoadsOutstanding())
	}
}

func TestLoadManyFailureOnTheSecondBatchKeepsTheFirstBatchsResults(t *testing.T) {
	ls := newLoadStation(t)
	ls.failFrame = 2
	c := openClient(t, ls)
	ords := ordsOf(70)
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err == nil {
		t.Fatal("no error for a refused second frame")
	}
	checkNodes(t, ords[:50], rs) // the ords of the batches done before the failure, no more
	if ls.loadFrames != 2 {
		t.Errorf("%d loadSlots frames, want 2 (a failure is not retried)", ls.loadFrames)
	}
	if c.LoadsOutstanding() != 0 {
		t.Errorf("%d loads outstanding, want 0: the first batch's loads came in, the second was rejected",
			c.LoadsOutstanding())
	}
}
