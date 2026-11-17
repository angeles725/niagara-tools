package writeops

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The batch form of n4_set_slot: each entry is planned and read back exactly as the single form
// (setSlotPlan, setSlotReadback); the batch concatenates their ops and inverses in entry order, so
// the engine journals and n4_rollback undoes them as one batch.

// setSlotBatch is a batch plan's private data: each entry and its own plan.
type setSlotBatch struct {
	entries []*pyjson.Object
	plans   []*plan.Planned
}

func entryName(i int) string { return fmt.Sprintf("writes[%d]", i) }

// setSlotBatchPlan plans every entry; one refused entry refuses the batch (its message names the
// entry), and so does a second entry for a (component, slot) an earlier one writes. The notes
// name their entry; incoming_links lists each entry's incoming links with the entry's ord as
// target_ord.
func setSlotBatchPlan(c plan.Client, entries []*pyjson.Object) (*plan.Planned, error) {
	out := &plan.Planned{Ops: []any{}, Inverse: []any{}, Notes: []any{}, Targets: pyjson.NewObject()}
	d, links, seen := &setSlotBatch{entries: entries}, []any{}, map[string]int{}
	for i, e := range entries {
		p, err := setSlotPlan(c, e)
		if err != nil {
			return nil, entryError(i, err)
		}
		key := fmt.Sprintf("%v\x00%s", get(p.Ops[0].(*pyjson.Object), "h"), str(e, "slot"))
		if j, dup := seen[key]; dup {
			return nil, plan.ToolError(fmt.Sprintf("%s sets slot %s of %s twice (%s already does); one entry per slot",
				entryName(i), pyjson.StrRepr(str(e, "slot")), str(e, "ord"), entryName(j)))
		}
		seen[key] = i
		out.Ops, out.Inverse = append(out.Ops, p.Ops...), append(out.Inverse, p.Inverse...)
		for _, n := range p.Notes {
			out.Notes = append(out.Notes, entryName(i)+" ("+str(e, "ord")+" "+str(e, "slot")+"): "+pyjson.Str(n))
		}
		for _, k := range p.Targets.Keys() {
			v, _ := p.Targets.Get(k)
			out.Targets.Set(k, v)
		}
		for _, l := range asList(get(p.Keys, "incoming_links")) {
			links = append(links, withTargetOrd(str(e, "ord"), l.(*pyjson.Object)))
		}
		d.plans = append(d.plans, p)
	}
	out.Keys, out.Data = obj("incoming_links", links), d
	return out, nil
}

// entryError prefixes a refusal with the entry it refuses; any other error is kept as it is.
func entryError(i int, err error) error {
	var te *jsonrpc.ToolError
	if errors.As(err, &te) {
		return plan.ToolError(entryName(i) + ": " + te.Msg)
	}
	return err
}

// withTargetOrd is an incoming link of an entry with the ORD it writes into first.
func withTargetOrd(ordText string, link *pyjson.Object) *pyjson.Object {
	out := obj("target_ord", ordText)
	for _, k := range link.Keys() {
		v, _ := link.Get(k)
		out.Set(k, v)
	}
	return out
}

// setSlotBatchReadback reads back each entry as the single form does: requested lists {ord, slot,
// value}, observed {ord, slot, requested, observed, verdict} per entry, and the batch is verified
// only when every entry is (else mismatch, the only other verdict an entry gives).
func setSlotBatchReadback(c plan.Client, d *setSlotBatch, replies []any) (any, any, any, string, error) {
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, e := range d.entries {
		r, _, o, v, err := setSlotReadback(c, e, d.plans[i], replies[i:i+1], nil)
		if err != nil {
			return fail(err)
		}
		requested = append(requested, obj("ord", str(e, "ord"), "slot", str(e, "slot"), "value", r))
		observed = append(observed, obj("ord", str(e, "ord"), "slot", str(e, "slot"), "requested", r,
			"observed", o, "verdict", v))
		if v != "verified" {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}
