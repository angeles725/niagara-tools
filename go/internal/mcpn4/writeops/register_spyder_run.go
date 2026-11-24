package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// registerRun runs the plan's follow_up on the device the add created, in the same batch: the
// steps are scope-checked, journaled as one component-intent before the first is sent, and
// stop at the first one the station (or the fault rule) refuses; that step is the read-back's
// failed_step and the batch keeps its inverse. Only a failure before any step was sent (the new
// device's ORD or handle, the journal) leaves the batch in-doubt.
func registerRun(b *plan.Batch, p *plan.Planned, replies []any) (*plan.Runs, error) {
	d := p.Data.(*registerData)
	name, _ := assignedName(p.Ops[0].(*pyjson.Object), replies[0]).(string)
	child, err := ord.ChildOrd(d.network, name)
	if err == nil {
		err = b.State.Scope.Check(child)
	}
	var h, pointsH any
	if err == nil {
		h, _, err = plan.Handle(b.Client, child, 1)
	}
	if err == nil && d.args.discover {
		pointsH, _, err = plan.Handle(b.Client, child+"/points", 1)
	}
	if err != nil {
		return nil, leftBehind(plan.InDoubtOf(b.ID, err), child)
	}
	d.child = child
	ops := d.ops(h, pointsH)
	if b.State.Journal.Append(obj("batch_id", b.ID, "ts", session.Now(), "phase", "component-intent", "ops", ops)) != nil {
		return nil, leftBehind(plan.InDoubtBecause(b.ID, "the component intent could not be written, nothing more was sent"), child)
	}
	for i, op := range ops {
		step := d.steps[i].(*pyjson.Object)
		label := pyjson.Str(get(step, "slot")) + pyjson.Str(get(step, "action"))
		if get(step, "action") == writeDeviceInstance { // the dry-run rule, on the device as it is now
			if _, nodes, err := plan.Handle(b.Client, child, 1); err == nil {
				if refused := instanceFaulted(child, nodes); refused != nil {
					d.failed = refused.Error()
					break
				}
			}
		}
		reply, err := sendStep(b.Client, op.(*pyjson.Object))
		if err != nil {
			d.failed = fmt.Sprintf("step %d of %d (%s) failed: %s; no later step was sent", i+1, len(ops), label, plan.PyStr(err))
			break
		}
		if get(step, "target") == "points" {
			d.discovery = reply
		}
	}
	return &plan.Runs{}, nil
}

// ops are the follow-up steps as station ops on the device handle h and its points extension's
// pointsH.
func (d *registerData) ops(h, pointsH any) []any {
	ops := make([]any, 0, len(d.steps))
	for _, s := range d.steps {
		step := s.(*pyjson.Object)
		switch {
		case get(step, "op") == "set":
			ops = append(ops, setOp(h, pyjson.Str(get(step, "slot")), simpleBSON(pyjson.Str(get(step, "type")), get(step, "value"))))
		case get(step, "target") == "points":
			ops = append(ops, invokeOp(obj("h", pointsH, "a", get(step, "action"))))
		default:
			ops = append(ops, invokeOp(obj("h", h, "a", get(step, "action"))))
		}
	}
	return ops
}

// sendStep sends one follow-up op: an invokeAction or a slot set.
func sendStep(c plan.Client, op *pyjson.Object) (any, error) {
	if _, ok := op.Get("ssc"); ok {
		return c.SSC("invokeAction", get(op, "arg"))
	}
	return c.Sync(op)
}

// leftBehind names the device the add created, so a stopped batch can be cleaned up.
func leftBehind(err *plan.InDoubt, child string) error {
	err.Msg += ". The add was applied: " + child + " exists (remove it to clean up)"
	return err
}
