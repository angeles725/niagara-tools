package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// rollbackReadbackAll is rollbackReadback, then the outgoing links the rollback re-created
// (bench retro 2026-11-15 delta 20): each must be on its target, from the restored source with
// the recorded slots. observed gains outgoing_links {restored, missing}; a missing one (never
// sent, or not on the target) makes the verdict mismatch, never verified.
func rollbackReadbackAll(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	requested, accepted, observed, verdict, err := rollbackReadback(c, args, p, replies, inverse)
	d := p.Data.(*rollbackData)
	if err != nil || len(d.outgoing) == 0 {
		return requested, accepted, observed, verdict, err
	}
	restored, missing := 0, append([]any{}, d.outgoingSkipped...)
	for _, s := range d.outgoingSent {
		sent := s.(*pyjson.Object)
		there, err := outgoingThere(c, sent)
		if err != nil {
			return fail(err)
		}
		if there {
			restored++
			continue
		}
		e := outgoingEntry(sent, "not on the target after the rollback")
		missing = append(missing, e)
	}
	o := observed.(*pyjson.Object)
	o.Set("outgoing_links", obj("restored", restored, "missing", missing))
	if len(missing) > 0 {
		o.Set("restored", false)
		verdict = "mismatch"
	}
	return requested, accepted, observed, verdict, nil
}

// outgoingThere reports whether the target of sent holds a link from source_h with its slots.
func outgoingThere(c plan.Client, sent *pyjson.Object) (bool, error) {
	_, nodes, err := plan.Handle(c, get(sent, "target").(string), 2)
	if err != nil {
		return false, err
	}
	want, err := handleOrd(get(sent, "source_h"))
	if err != nil {
		return false, err
	}
	for _, name := range links(nodes) {
		f := linkSlots(nodes, name)
		tail, err := sourceTail(f)
		if err == nil && tail == want && box.PyEqual(get(f, "sourceSlotName"), get(sent, "source_slot")) &&
			box.PyEqual(get(f, "targetSlotName"), get(sent, "target_slot")) {
			return true, nil
		}
	}
	return false, nil
}
