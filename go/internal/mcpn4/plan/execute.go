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

// execute is _execute: send the journaled batch, run its follow-up ops, read it back and
// journal the result. The inverse, recover and read-back steps catch every failure (a panic
// too), as Python's `except Exception` does: the write already happened.
func (e *Engine) execute(c Client, impl Impl, args *pyjson.Object, planned *Planned, batchID string) (*pyjson.Object, error) {
	replies, err := e.sendAll(c, impl, args, planned, batchID)
	if err != nil {
		return nil, err
	}
	out := obj("dry_run", false, "batch_id", batchID)
	runs := &Runs{}
	if impl.Run != nil { // a rollback's nested components and links: same batch, same guards
		if runs, err = impl.Run(&Batch{Client: c, State: e.State, ID: batchID}, planned, replies); err != nil {
			return nil, err
		}
	}
	if len(runs.Frozen) > 0 {
		out.Set("frozen_children_not_restored", runs.Frozen)
	}
	withKeys(out, planned, []string{"snapshot_truncated"}) // what a remove's snapshot could not keep
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
			e.scrubGaps(rb.observed)
			set(out, "requested", rb.requested, "accepted", rb.accepted, "observed", rb.observed, "verdict", rb.verdict)
		}
	}
	inverse = append(list(runs.Inverse), inverse...) // links first: undoing must precede removing their ends
	out.Set("inverse", inverse)
	if o, ok := get(out, "observed").(*pyjson.Object); ok { // promote the headline evidence of a tool
		for _, k := range []string{"persisted", "evidence", "backup", "relinks", "frozen_config_not_restored",
			"link_inputs_not_restored"} {
			if v, ok := o.Get(k); ok {
				out.Set(k, v)
			}
		}
	}
	if runs.Doubt {
		out.Set("in_doubt", true)
	}
	result := obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", get(out, "accepted"),
		"inverse", inverse, "verdict", get(out, "verdict"))
	if runs.Report != nil {
		result.Set("relinks", runs.Report)
	}
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

// scrubGaps is _scrub_gap_errors: defence in depth, the station read errors a read-back put in
// its frozen-child gaps are scrubbed.
func (e *Engine) scrubGaps(observed any) {
	o, _ := observed.(*pyjson.Object)
	if o == nil {
		return
	}
	gaps, _ := get(o, "frozen_config_not_restored").([]any)
	for _, g := range gaps {
		if gap, ok := g.(*pyjson.Object); ok {
			if why, ok := gap.Get("readback_error"); ok {
				gap.Set("readback_error", e.Ctx.Scrub(pyjson.Str(why)))
			}
		}
	}
}

func canonical(v any) string {
	b, _ := pyjson.Marshal(v, pyjson.Canonical) // decoded values always encode
	return string(b)
}

// inverse is the inverse a batch journals: the planned one, or the one built from the replies
// (a nil one is written as [], _inverse_list).
func (impl Impl) inverse(planned *Planned, replies []any) ([]any, error) {
	if impl.Inverse == nil {
		return list(planned.Inverse), nil
	}
	return impl.Inverse(planned, replies)
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
