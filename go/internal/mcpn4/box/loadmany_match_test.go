package box

import "testing"

// A load op carries the handle of the component it loaded ("h", SyncOp.encode), never its ord, so
// an ord whose handle is not cached yet is matched by arrival order (BBoxService answers a
// frame's messages of one session in order, into a FIFO of loads). A cached handle is checked
// against the op, and so is an op of the handle of another ord of the batch.

func TestLoadManyMixedCachedAndUncachedBatchInOrderIsOneFrame(t *testing.T) {
	ls := newLoadStation(t)
	c := openClient(t, ls)
	ords := ordsOf(4)
	c.SetHandle(ords[1], "h:"+ords[1])
	c.SetHandle(ords[3], "h:"+ords[3])
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords, rs)
	if ls.loadFrames != 1 || c.LoadsOutstanding() != 0 {
		t.Errorf("%d loadSlots frames, %d outstanding; want 1 and 0", ls.loadFrames, c.LoadsOutstanding())
	}
}

func TestLoadManyReorderedOpsOfAMixedBatchNeverGiveAWrongTree(t *testing.T) {
	for name, order := range map[string][]int{
		"cached op first":                     {1, 0, 2, 3},
		"cached op last":                      {0, 2, 3, 1},
		"cached ops swapped":                  {0, 3, 2, 1},
		"uncached op in the middle of cached": {1, 2, 3, 0},
	} {
		ls := newLoadStation(t)
		ls.reorder = func(ops []any) []any {
			if len(ops) != 4 {
				return ops
			}
			out := make([]any, 4)
			for i, j := range order {
				out[i] = ops[j]
			}
			return out
		}
		c := openClient(t, ls)
		ords := ordsOf(4)
		c.SetHandle(ords[1], "h:"+ords[1])
		c.SetHandle(ords[3], "h:"+ords[3])
		rs, err := c.LoadMany(ords, 1, LoadOptions{})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		checkNodes(t, ords, rs) // each ord has its own tree, loaded alone where the batch could not tell
		if ls.loadFrames < 2 {
			t.Errorf("%s: %d loadSlots frames: the mismatch was not detected", name, ls.loadFrames)
		}
	}
}

func TestLoadManyOfOnlyUncachedOrdsKeepsRequestOrder(t *testing.T) {
	ls := newLoadStation(t)
	c := openClient(t, ls)
	ords := ordsOf(5)
	rs, err := c.LoadMany(ords, 1, LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	checkNodes(t, ords, rs)
	if ls.loadFrames != 1 {
		t.Errorf("%d loadSlots frames, want 1", ls.loadFrames)
	}
}
