package box

import (
	"fmt"
	"math"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// PollDelays is box.poll_delays: the sleeps between attempts polls: first, doubling, each at
// most limit, their sum at most total, each rounded to 6 decimals. No sleep follows the last
// poll. (0.1, 0.8, 3.0, 20) gives 0.1, 0.2, 0.4, 0.8, 0.8, 0.7.
func PollDelays(first, limit, total float64, attempts int) []float64 {
	var out []float64
	slept, delay := 0.0, first
	for range max(0, attempts-1) {
		step := round6(math.Min(math.Min(delay, limit), total-slept))
		if step <= 0 {
			break
		}
		out = append(out, step)
		slept += step
		delay *= 2
	}
	return out
}

// round6 is round(x, 6): correctly rounded from the exact binary value, as CPython rounds.
func round6(x float64) float64 {
	f, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 6, 64), 64)
	return f
}

// PendingEvents returns the events load_tree set aside (a copy); at most MaxPendingEvents.
func (c *Client) PendingEvents() []any { return append([]any{}, c.pending...) }

// DrainPendingEvents returns the pending events and clears them.
func (c *Client) DrainPendingEvents() []any {
	out := c.pending
	if out == nil {
		out = []any{}
	}
	c.pending = nil
	return out
}

// DroppedEvents counts events discarded because the pending list was full.
func (c *Client) DroppedEvents() int { return c.dropped }

// LoadsOutstanding counts loadSlots requests sent minus load ops seen.
func (c *Client) LoadsOutstanding() int { return c.loadsOutstanding }

func (c *Client) keep(event any) {
	c.pending = append(c.pending, event)
	for len(c.pending) > MaxPendingEvents {
		c.pending = c.pending[1:]
		c.dropped++
	}
}

// keepRest keeps event minus its consumed ops (all of it when ops is nil).
func (c *Client) keepRest(event any, ops []any) {
	evs, _ := get(event, "evs")
	evsObj, isObj := evs.(*pyjson.Object)
	if ops == nil || !isObj {
		c.keep(event)
	} else if len(ops) > 0 {
		c.keep(with(event.(*pyjson.Object), "evs", with(evsObj, "ops", ops)))
	}
}

func isLoadOp(op any) bool {
	nm, _ := get(op, "nm")
	return nm == "l"
}

func (c *Client) settle(op any) {
	if isLoadOp(op) {
		c.loadsOutstanding = max(0, c.loadsOutstanding-1)
	}
}

// split is BoxClient._split: the first l op whose handle matches (any when handle is nil) and
// whether a load op of another handle was seen. Every other event and op is kept pending.
func (c *Client) split(events []any, handle any) (*pyjson.Object, bool) {
	var found *pyjson.Object
	sawOther := false
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
			_, bIsObj := b.(*pyjson.Object)
			h, _ := get(op, "h")
			loadWithBody := isLoadOp(op) && bIsObj
			if found == nil && loadWithBody && (handle == nil || pyEqual(h, handle)) {
				found = op.(*pyjson.Object)
			} else {
				if handle != nil && loadWithBody && !pyEqual(h, handle) {
					sawOther = true
				}
				rest = append(rest, op)
			}
			c.settle(op)
		}
		c.keepRest(event, rest)
	}
	return found, sawOther
}

// loadOnce is BoxClient._load_once: request ord and poll up to attempts times for its load op
// (nil when none came). With cached (the handle came from the cache) a load op of another
// handle proves the cached one stale and ends the wait, but only when no earlier loadSlots
// request was still unanswered.
func (c *Client) loadOnce(ord string, depth, attempts int, delay float64, handle any, cached bool) (*pyjson.Object, error) {
	events, err := c.Poll() // queued before the request: cannot be its reply
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		c.keep(event)
		evs, _ := get(event, "evs")
		if _, ok := evs.(*pyjson.Object); !ok {
			continue
		}
		raw, _ := get(evs, "ops")
		if !truthy(raw) {
			continue
		}
		ops, err := pyIter(raw)
		if err != nil {
			return nil, err
		}
		for _, op := range ops {
			c.settle(op)
		}
	}
	trusted := c.loadsOutstanding == 0
	c.loadsOutstanding++
	if _, err := c.SSC("loadSlots", obj("o", ord, "d", depth)); err != nil {
		c.loadsOutstanding = max(0, c.loadsOutstanding-1) // rejected or never sent
		return nil, err
	}
	delays := PollDelays(delay, MaxPollDelay, c.LoadWait, attempts)
	for i := 0; ; i++ {
		events, err := c.Poll()
		if err != nil {
			return nil, err
		}
		op, sawOther := c.split(events, handle)
		if op != nil {
			c.loadsOutstanding = 0 // replies come in request order: earlier ones are void
			return op, nil
		}
		if cached && trusted && sawOther {
			return nil, nil
		}
		if i >= len(delays) {
			return nil, nil
		}
		c.Sleep(delays[i])
	}
}

// LoadOptions are load_tree's polling arguments. Zero Attempts takes enough polls to fill the
// LoadWait window, zero Delay takes FirstPollDelay, nil Handle means unknown.
type LoadOptions struct {
	Attempts int
	Delay    float64
	Handle   any
}

// LoadTree is BoxClient.load_tree: load ord and return the flat {path: node} dict (the root
// key is ""). Only the load op for the requested target is taken; every other event or op is
// kept pending and two load ops are never merged. A handle taken from the cache that gets no
// matching op is dropped and the request repeated once with the handle unknown.
func (c *Client) LoadTree(ord string, depth int, opt LoadOptions) (*pyjson.Object, error) {
	attempts, delay := opt.Attempts, opt.Delay
	if attempts == 0 { // 12 for the default 3 s window
		attempts = max(12, int(math.Ceil(c.LoadWait/MaxPollDelay))+4)
	}
	if delay == 0 {
		delay = FirstPollDelay
	}
	explicit := opt.Handle != nil
	handle := opt.Handle
	if !explicit {
		handle, _ = c.handles.get(ord)
	}
	op, err := c.loadOnce(ord, depth, attempts, delay, handle, handle != nil && !explicit)
	if err != nil {
		return nil, err
	}
	if op == nil && handle != nil && !explicit {
		c.handles.drop(func(k string) bool { return k == ord })
		if op, err = c.loadOnce(ord, depth, attempts, delay, nil, false); err != nil {
			return nil, err
		}
	}
	if op == nil {
		return nil, boxError(fmt.Sprintf("no load event for %s after %d polls", ord, attempts), Channel, "loadSlots")
	}
	b, _ := op.Get("b")
	nodes := pyjson.NewObject()
	if err := flatten(b.(*pyjson.Object), "", nodes); err != nil {
		return nil, err
	}
	root, _ := nodes.Get("")
	if h, _ := get(root, "h"); truthy(h) {
		c.handles.set(ord, h)
	}
	return nodes, nil
}

// flatten is box._flatten: node under path, then each child (its "n" joined with "/").
// A malformed tree fails as the Python walk does: TypeError on a non-iterable "s", a child that
// is no dict or a name that is no string, KeyError on a child without "n".
func flatten(node *pyjson.Object, path string, out *pyjson.Object) error {
	out.Set(path, node)
	raw, ok := node.Get("s")
	if !ok {
		return nil
	}
	kids, err := pyIter(raw)
	if err != nil {
		return err
	}
	for _, kid := range kids {
		child, ok := kid.(*pyjson.Object)
		if !ok {
			return &PyError{"TypeError"}
		}
		n, ok := child.Get("n")
		if !ok {
			return &PyError{"KeyError"}
		}
		name, ok := n.(string)
		if !ok {
			return &PyError{"TypeError"}
		}
		if path != "" {
			name = path + "/" + name
		}
		if err := flatten(child, name, out); err != nil {
			return err
		}
	}
	return nil
}
