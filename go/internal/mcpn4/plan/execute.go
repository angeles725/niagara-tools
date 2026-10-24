package plan

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// readback is what Impl.Readback (or Impl.Recover: observed and inverse) returns.
type readback struct {
	requested, accepted, observed any
	verdict                       string
	inverse                       []any
}

// execute is _execute: send the journaled batch, read it back and journal the result. The
// inverse, recover and read-back steps catch every failure (a panic too), as Python's
// `except Exception` does: the write already happened.
func (e *Engine) execute(c Client, impl Impl, args *pyjson.Object, planned *Planned, batchID string) (*pyjson.Object, error) {
	replies := []any{}
	for _, op := range planned.Ops {
		reply, err := send(c, op)
		if err != nil { // one-op batches: the intent without a result already marks it in-doubt
			return nil, inDoubt(batchID, err)
		}
		replies = append(replies, reply)
	}
	out := obj("dry_run", false, "batch_id", batchID)
	var accepted any
	if len(replies) > 0 {
		accepted = replies[0]
	}
	inverse, err := jsonrpc.Recover(func() ([]any, error) { return impl.inverse(planned, replies) })
	if err != nil { // the station's reply cannot be trusted: outcome unknown
		why := PyType(err)
		var be *box.Error
		if errors.As(err, &be) {
			why = be.Msg
		}
		var observed any
		inverse = []any{}
		if impl.Recover != nil {
			found, err := jsonrpc.Recover(func() (readback, error) {
				o, inv, err := impl.Recover(c, args, planned, replies)
				return readback{observed: o, inverse: inv}, err
			})
			if err == nil {
				observed, inverse = found.observed, list(found.inverse)
			}
		}
		set(out, "requested", nil, "accepted", accepted, "observed", observed, "verdict", "unverified",
			"in_doubt", true, "readback_error", e.Ctx.Scrub(why))
	} else {
		rb, err := jsonrpc.Recover(func() (readback, error) {
			r, a, o, v, err := impl.Readback(c, args, planned, replies, inverse)
			return readback{requested: r, accepted: a, observed: o, verdict: v}, err
		})
		if err != nil { // whatever the read-back hits, the write already happened
			why := PyType(err) + ": " + PyStr(err)
			var be *box.Error
			if errors.As(err, &be) {
				why = be.Msg
			}
			set(out, "requested", nil, "accepted", accepted, "observed", nil, "verdict", "failed",
				"readback_failed", true, "readback_error", e.Ctx.Scrub(why))
		} else {
			set(out, "requested", rb.requested, "accepted", rb.accepted, "observed", rb.observed, "verdict", rb.verdict)
		}
	}
	out.Set("inverse", inverse)
	if o, ok := get(out, "observed").(*pyjson.Object); ok { // promote the headline evidence of a tool
		for _, k := range []string{"persisted", "evidence", "relinks", "frozen_config_not_restored",
			"link_inputs_not_restored"} {
			if v, ok := o.Get(k); ok {
				out.Set(k, v)
			}
		}
	}
	result := obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", get(out, "accepted"),
		"inverse", inverse, "verdict", get(out, "verdict"))
	for _, k := range []string{"readback_failed", "in_doubt"} { // accepted by the station, outcome unverified
		if _, ok := out.Get(k); ok {
			result.Set(k, true)
		}
	}
	if e.State.Journal.Append(result) != nil {
		ops, _ := pyjson.Marshal(inverse, pyjson.Canonical)
		out.Set("warnings", []any{"the journal result could not be written: batch " + batchID +
			" is in-doubt; inverse ops: " + string(ops)})
	}
	return out, nil
}

// inverse is the inverse a batch journals: the planned one, or the one built from the replies
// (a nil one is written as [], _inverse_list).
func (impl Impl) inverse(planned *Planned, replies []any) ([]any, error) {
	if impl.Inverse == nil {
		return list(planned.Inverse), nil
	}
	return impl.Inverse(planned, replies)
}

// send is _send: an ssc op or one sync op; a removal makes every cached handle stale.
func send(c Client, op any) (any, error) {
	o, _ := op.(*pyjson.Object) // every planned op is a dict
	if key, ok := o.Get("ssc"); ok {
		arg, _ := o.Get("arg")
		return c.SSC(pyjson.Str(key), arg)
	}
	reply, err := c.Sync(op)
	if nm, _ := o.Get("nm"); err == nil && nm == "v" {
		c.ClearHandles()
	}
	return reply, err
}

func list(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}

func get(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

// set is out.update(k=v, ...) in argument order.
func set(o *pyjson.Object, kv ...any) {
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
}
