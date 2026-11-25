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

// LoadMany is LoadTree of every ord at depth, with the loadSlots requests in one frame per
// MaxFrameMessages ords (CallMany) instead of one frame each: poll, loadSlots frame(s), then
// polls until every load op is in. A load is an event, not a reply, so the ops are matched to the
// ords in request order (the station answers a frame's messages in order, one session, the order
// LoadTree relies on too) and checked against a cached handle where there is one. An ord the
// station refused is its LoadResult.Err and never fails the others. An ord whose op does not match
// or never comes is loaded on its own by LoadTree (its retries and its error). The call fails
// (nil, err) on a transport, auth or malformed-reply failure, as LoadTree does.
// Fewer than two ords, or a load still outstanding from an earlier request (its late op would
// be mistaken for one of these), is plain LoadTree per ord.
func (c *Client) LoadMany(ords []string, depth int, opt LoadOptions) ([]LoadResult, error) {
	out := make([]LoadResult, len(ords))
	if len(ords) >= 2 {
		if err := c.settleStale(); err != nil {
			return nil, err
		}
	}
	if len(ords) < 2 || c.loadsOutstanding != 0 {
		for i := range ords {
			if err := c.loadAlone(ords[i], depth, opt, &out[i]); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	want, err := c.requestLoads(ords, depth, out)
	if err != nil {
		return nil, err
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
			return nil, err
		}
		for _, op := range c.takeLoads(events, len(want)-next) {
			idx := want[next]
			if h, ok := c.handles.get(ords[idx]); ok && !PyEqual(h, opField(op, "h")) {
				stop = true // a cached handle that is not this op's: do not guess
				break
			}
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
			return nil, err
		}
	}
	return out, nil
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

// requestLoads sends the loadSlots of every ord (once more after a timeout, as loadSlotsRequest
// does) and returns the indexes of the ords the station accepted; a refused one is out[i].Err.
func (c *Client) requestLoads(ords []string, depth int, out []LoadResult) ([]int, error) {
	args := make([]any, len(ords))
	for i, o := range ords {
		args[i] = obj("o", o, "d", depth)
	}
	c.loadsOutstanding += len(ords)
	replies, err := c.SSCMany("loadSlots", args)
	if c.retryRead(err) {
		c.loadsOutstanding += len(ords) // a timeout does not prove the first request was dropped
		replies, err = c.SSCMany("loadSlots", args)
	}
	if err != nil {
		if !timedOut(err) {
			c.loadsOutstanding = max(0, c.loadsOutstanding-len(ords)) // rejected or never sent
		}
		return nil, err
	}
	var want []int
	for i, r := range replies {
		if r.Err != nil {
			out[i].Err = r.Err
			c.loadsOutstanding = max(0, c.loadsOutstanding-1)
			continue
		}
		want = append(want, i)
	}
	return want, nil
}

// settleStale polls once and keeps what came: it was queued before the request, so it is no
// reply to it; its load ops settle the outstanding count (the start of loadOnce).
func (c *Client) settleStale() error {
	events, err := c.Poll()
	if err != nil {
		return err
	}
	for _, event := range events {
		c.keep(event)
		evs, _ := get(event, "evs")
		raw, _ := get(evs, "ops")
		ops, _ := raw.([]any)
		for _, op := range ops {
			c.settle(op)
		}
	}
	return nil
}

// takeLoads is split for several ops: the first limit load ops that carry a body, in arrival
// order; every other event and op is kept pending.
func (c *Client) takeLoads(events []any, limit int) []*pyjson.Object {
	var found []*pyjson.Object
	for _, event := range events {
		evs, _ := get(event, "evs")
		raw, _ := get(evs, "ops")
		ops, ok := raw.([]any)
		if !ok {
			c.keepRest(event, nil)
			continue
		}
		rest := []any{}
		for _, op := range ops {
			b, _ := get(op, "b")
			if _, isObj := b.(*pyjson.Object); isObj && isLoadOp(op) && len(found) < limit {
				found = append(found, op.(*pyjson.Object))
			} else {
				rest = append(rest, op)
			}
			c.settle(op)
		}
		c.keepRest(event, rest)
	}
	return found
}

// loadedNodes is the end of LoadTree: the load op's tree flattened, its root handle remembered.
func (c *Client) loadedNodes(ord string, op *pyjson.Object) (*pyjson.Object, error) {
	b, _ := op.Get("b")
	nodes := pyjson.NewObject()
	if err := flatten(b.(*pyjson.Object), "", nodes); err != nil {
		return nil, err
	}
	root, _ := nodes.Get("")
	if h, _ := get(root, "h"); pyjson.Truthy(h) {
		c.handles.set(ord, h)
	}
	return nodes, nil
}
