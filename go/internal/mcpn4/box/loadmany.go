package box

import (
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// LoadResult is one ord of LoadMany: its flat {path: node} tree, or the error that ord failed with.
type LoadResult struct {
	Nodes *pyjson.Object
	Err   error
}

// LoadMany is LoadTree of every ord at depth, with the loadSlots requests of up to MaxFrameMessages
// ords in one frame (CallMany) instead of one frame each: poll, loadSlots frame, then polls until
// every load op of the frame is in. More ords go in batches, one frame per poll cycle and each
// batch after the one before has come in, so a timed-out frame is asked again alone, never with
// the frames before it. A load is an event, not a reply: its op names the component loaded by
// handle ("h"), not by ord (BBoxService.handleMessages answers a frame's messages of a session in
// order, ProxyBroker.loadOp queues them FIFO: the order LoadTree relies on too). So the ops are
// matched to the ords in request order, and checked against the handle of every ord whose handle
// is cached (matchesOp). An ord the station refused is its LoadResult.Err and never fails the
// others. An ord whose op does not match or never comes is loaded on its own by LoadTree (its
// retries and its error). The call fails on a transport, auth or malformed-reply failure, as
// LoadTree does: (nil, err) when the first batch failed, else the results of the batches done
// before the failure (a prefix of ords) and err. Fewer than two ords in a batch, or a load still
// outstanding from an earlier request (its late op would be mistaken for one of these), is plain
// LoadTree per ord.
func (c *Client) LoadMany(ords []string, depth int, opt LoadOptions) ([]LoadResult, error) {
	out := make([]LoadResult, len(ords))
	limit := c.FrameCap
	if limit <= 0 {
		limit = MaxFrameMessages
	}
	for start := 0; start < len(ords); start += limit {
		end := min(start+limit, len(ords))
		if err := c.loadBatch(ords[start:end], depth, opt, out[start:end], start == 0); err != nil {
			if start == 0 {
				return nil, err
			}
			return out[:start], err
		}
	}
	return out, nil
}

// loadBatch is LoadMany of one frame's ords into out. Only the first batch polls for stale loads
// first: after a batch every op it asked for is in, or loadsOutstanding says otherwise.
func (c *Client) loadBatch(ords []string, depth int, opt LoadOptions, out []LoadResult, first bool) error {
	if first && len(ords) >= 2 {
		if err := c.settleStale(); err != nil {
			return err
		}
	}
	if len(ords) < 2 || c.loadsOutstanding != 0 {
		for i := range ords {
			if err := c.loadAlone(ords[i], depth, opt, &out[i]); err != nil {
				return err
			}
		}
		return nil
	}
	want, err := c.requestLoads(ords, depth, out)
	if err != nil {
		return err
	}
	attempts, delay := opt.Attempts, opt.Delay
	if attempts == 0 {
		attempts = max(12, int(math.Ceil(c.LoadWait/MaxPollDelay))+4)
	}
	if delay == 0 {
		delay = FirstPollDelay
	}
	delays := PollDelays(delay, MaxPollDelay, c.LoadWait, attempts)
	next, stop := 0, false // want[next:] still wait for their op
	for i := 0; next < len(want); i++ {
		events, err := c.Poll()
		if err != nil {
			return err
		}
		for _, op := range c.takeLoads(events, len(want)-next) {
			if !c.matchesOp(ords, want[next:], op) {
				stop = true // an op of another ord's handle: do not guess
				break
			}
			idx := want[next]
			out[idx].Nodes, out[idx].Err = c.loadedNodes(ords[idx], op)
			next++
		}
		if stop || i >= len(delays) {
			break
		}
		if next < len(want) {
			c.Sleep(delays[i])
		}
	}
	for _, idx := range want[next:] {
		if err := c.loadAlone(ords[idx], depth, opt, &out[idx]); err != nil {
			return err
		}
	}
	return nil
}

// matchesOp is whether op is the load of ords[pending[0]], pending being the ords still waiting
// for theirs. Its handle must be the cached handle of that ord; an ord with none cached takes the
// next op in order, unless the op is the load of the cached handle of another pending ord (the
// stream is out of order). Two uncached ords swapped cannot be told apart: the station sends no
// ord, only the handle.
func (c *Client) matchesOp(ords []string, pending []int, op *pyjson.Object) bool {
	h := opField(op, "h")
	if known, ok := c.handles.get(ords[pending[0]]); ok {
		return PyEqual(known, h)
	}
	for _, idx := range pending[1:] {
		if known, ok := c.handles.get(ords[idx]); ok && PyEqual(known, h) {
			return false
		}
	}
	return true
}

// opField is o.get(k) of an op.
func opField(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

// loadAlone is LoadTree into res; a failure other than an AuthError (which stops the batch: more
// rejected logins count toward the lockout) is that ord's result.
func (c *Client) loadAlone(ord string, depth int, opt LoadOptions, res *LoadResult) error {
	var err error
	res.Nodes, err = c.LoadTree(ord, depth, opt)
	if e, ok := err.(*Error); ok && e.IsAuth() {
		return err
	}
	res.Err = err
	return nil
}
