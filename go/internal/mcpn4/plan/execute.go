package plan

import (
	"errors"
	"fmt"

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
	replies, err := e.sendAll(c, impl, planned, batchID)
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
		for _, k := range []string{"persisted", "evidence", "relinks", "frozen_config_not_restored",
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

// sendAll is _send_all: the batch's ops in order; a failure at op k is its partial in-doubt
// error.
func (e *Engine) sendAll(c Client, impl Impl, planned *Planned, batchID string) ([]any, error) {
	replies := []any{}
	for _, op := range planned.Ops {
		reply, err := send(c, op)
		if err != nil {
			return nil, e.inDoubtPartial(impl, planned, batchID, replies, err)
		}
		replies = append(replies, reply)
	}
	return replies, nil
}

// inDoubtPartial is _in_doubt_partial: op k of a multi-op batch failed after ops 0..k-1 were
// accepted. Their replies and inverse and an in-doubt result record go to the journal, so the
// batch never looks completed; a one-op batch keeps the plain in-doubt error (its intent
// without a result says everything).
func (e *Engine) inDoubtPartial(impl Impl, planned *Planned, batchID string, replies []any, why error) error {
	err := inDoubt(batchID, why)
	total, sent := len(planned.Ops), len(replies)
	if total == 1 {
		return err
	}
	head := *planned
	head.Ops = planned.Ops[:sent]
	inverse, ierr := jsonrpc.Recover(func() ([]any, error) { return impl.inverse(&head, replies) })
	shown := "unknown" // an inverse cannot be built from these replies: say so
	if inverse = list(inverse); ierr == nil {
		shown = canonical(inverse)
	}
	done := "no op before it was sent"
	switch {
	case sent == 1:
		done = "op 1 was sent"
	case sent > 1:
		done = fmt.Sprintf("ops 1-%d were sent", sent)
	}
	err.Msg = fmt.Sprintf("%s; op %d of %d failed: %s and accepted by the station (replies: %s), op %d may or may not "+
		"have been applied and no later op was sent. Inverse of the sent ops: %s", err.Msg, sent+1, total, done,
		canonical(replies), sent+1, shown)
	if ierr != nil {
		inverse = []any{}
	}
	if e.State.Journal.Append(obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", replies,
		"inverse", inverse, "verdict", "unverified", "in_doubt", true, "sent", sent, "failed_op", sent)) != nil {
		err.Msg += " (the in-doubt result record could not be journaled)"
	}
	return err
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
