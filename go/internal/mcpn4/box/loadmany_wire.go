package box

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

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
