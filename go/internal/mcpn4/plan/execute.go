package plan

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// execute is _execute: send the journaled batch, read it back and journal the result.
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
	inverse := list(planned.Inverse)
	requested, acc, observed, verdict, err := impl.Readback(c, args, planned, replies)
	if err != nil { // whatever the read-back hits, the write already happened
		why := PyType(err) + ": " + PyStr(err)
		var be *box.Error
		if errors.As(err, &be) {
			why = be.Msg
		}
		out.Set("requested", nil)
		out.Set("accepted", accepted)
		out.Set("observed", nil)
		out.Set("verdict", "failed")
		out.Set("readback_failed", true)
		out.Set("readback_error", e.Ctx.Scrub(why))
	} else {
		out.Set("requested", requested)
		out.Set("accepted", acc)
		out.Set("observed", observed)
		out.Set("verdict", verdict)
	}
	out.Set("inverse", inverse)
	if o, ok := observed.(*pyjson.Object); ok { // promote the headline evidence of a tool
		for _, k := range []string{"persisted", "evidence", "relinks", "frozen_config_not_restored",
			"link_inputs_not_restored"} {
			if v, ok := o.Get(k); ok {
				out.Set(k, v)
			}
		}
	}
	acc, _ = out.Get("accepted")
	v, _ := out.Get("verdict")
	result := obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", acc,
		"inverse", inverse, "verdict", v)
	if _, failed := out.Get("readback_failed"); failed { // accepted by the station, outcome unverified
		result.Set("readback_failed", true)
	}
	if e.State.Journal.Append(result) != nil {
		ops, _ := pyjson.Marshal(inverse, pyjson.Canonical)
		out.Set("warnings", []any{"the journal result could not be written: batch " + batchID +
			" is in-doubt; inverse ops: " + string(ops)})
	}
	return out, nil
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
