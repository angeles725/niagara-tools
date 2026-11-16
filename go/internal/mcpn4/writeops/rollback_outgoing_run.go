package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// outgoingOps are the checkLinks ops that re-create the recorded outgoing links of each restored
// component (sent with the relinks by sendLinks), with each op's target handle. A link whose
// target is gone, was replaced or left the scope, or whose source was not re-created, is not
// sent: d.outgoingSkipped holds it with the reason, d.outgoingSent what was sent.
func outgoingOps(b *plan.Batch, p *plan.Planned, d *rollbackData, base map[int]string, replies []any) (ops, targets []any, err error) {
	for i, op := range p.Ops {
		o := op.(*pyjson.Object)
		if get(o, "nm") != "a" {
			continue
		}
		comp, err := restoredOrd(d, o, replies[i])
		if err != nil {
			return nil, nil, err
		}
		nodes, err := b.Client.LoadTree(comp, SnapshotDepth, box.LoadOptions{})
		if err != nil { // the components exist, their links were never attempted
			return nil, nil, withCreated(plan.InDoubtOf(b.ID, err), base, d.created)
		}
		for _, s := range d.outgoing {
			spec := s.(*pyjson.Object)
			srcH, tgtH, reason := outgoingEndsNow(b, comp, nodes, spec)
			if reason != "" {
				d.outgoingSkipped = append(d.outgoingSkipped, outgoingEntry(spec, reason))
				continue
			}
			ops = append(ops, linkOp(srcH, get(spec, "source_slot"), tgtH, get(spec, "target_slot")))
			targets = append(targets, tgtH)
			sent := outgoingEntry(spec, "")
			sent.Set("source_h", srcH)
			d.outgoingSent = append(d.outgoingSent, sent)
		}
	}
	return ops, targets, nil
}

// outgoingEndsNow are the handles of a link's source (in the restored component's nodes) and
// target, or why it cannot be sent.
func outgoingEndsNow(b *plan.Batch, comp string, nodes, spec *pyjson.Object) (src, tgt any, reason string) {
	if outgoingEnds(b.State.Scope, comp, spec) != nil {
		return nil, nil, "an end is outside the write scope"
	}
	srcNode := box.Node(nodes, get(spec, "source_path").(string))
	if srcNode == nil || !pyjson.Truthy(get(srcNode, "h")) {
		return nil, nil, "source not re-created"
	}
	tgt, _, err := plan.Handle(b.Client, get(spec, "target").(string), 1)
	if err != nil || !pyjson.Truthy(tgt) {
		return nil, nil, "target not found"
	}
	if was, ok := spec.Get("target_h"); ok && !box.PyEqual(was, tgt) {
		return nil, nil, "target replaced since the remove (handle " + pyjson.Str(was) + ", now " + pyjson.Str(tgt) + ")"
	}
	return get(srcNode, "h"), tgt, ""
}

// outgoingEntry is an outgoing link as the read-back reports it, with reason when not "".
func outgoingEntry(spec *pyjson.Object, reason string) *pyjson.Object {
	e := obj("target", get(spec, "target"), "target_slot", get(spec, "target_slot"),
		"source_path", get(spec, "source_path"), "source_slot", get(spec, "source_slot"))
	if reason != "" {
		e.Set("reason", reason)
	}
	return e
}
