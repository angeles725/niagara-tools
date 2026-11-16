package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// runRelinks is _run_relinks: the links of the restored components, both ends in scope and
// present, sent by sendLinks. Its report counts the relinks restored, skipped and ambiguous.
func runRelinks(b *plan.Batch, p *plan.Planned, d *rollbackData, base map[int]string, replies []any, runs *plan.Runs) error {
	ops, targets, restoredComps, err := relinkOps(b, p, d, base, replies)
	if err != nil {
		return err
	}
	runs.Inverse = []any{}
	states, err := sendLinks(b, d, base, ops, targets, runs)
	if err != nil {
		return err
	}
	restored, ambiguous := tally(states)
	runs.Report = obj("restored", restored, "skipped", len(d.relinks)*restoredComps-restored)
	if ambiguous > 0 {
		runs.Report.Set("ambiguous", ambiguous)
	}
	runs.Doubt, d.report = ambiguous > 0, runs.Report
	return nil
}

// relinkOps are the checkLinks ops of the relinks whose ends exist and are in scope, with each
// op's target handle and the number of restored components; a skipped relink is recorded in
// d.skipped.
func relinkOps(b *plan.Batch, p *plan.Planned, d *rollbackData, base map[int]string, replies []any) (ops, targets []any,
	restoredComps int, err error) {
	d.skipped = []any{}
	for i, op := range p.Ops {
		o := op.(*pyjson.Object)
		if get(o, "nm") != "a" {
			continue
		}
		restoredComps++
		comp, err := restoredOrd(d, o, replies[i])
		if err != nil {
			return nil, nil, 0, err
		}
		nodes, err := b.Client.LoadTree(comp, SnapshotDepth, box.LoadOptions{})
		if err != nil { // the components exist, their links were never attempted
			return nil, nil, 0, withCreated(plan.InDoubtOf(b.ID, err), base, d.created)
		}
		for _, s := range d.relinks {
			spec := s.(*pyjson.Object)
			src, tgt := box.Node(nodes, get(spec, "source_path").(string)), box.Node(nodes, get(spec, "target_path").(string))
			var se *safety.Error
			if err := checkEnds(b.State.Scope, comp, spec); errors.As(err, &se) {
				src = nil
			} else if err != nil {
				return nil, nil, 0, err
			}
			if src == nil || tgt == nil || !pyjson.Truthy(get(src, "h")) || !pyjson.Truthy(get(tgt, "h")) {
				entry, err := inputEntry(comp, spec, "relink skipped") // an end is missing or out of scope
				if err != nil {
					return nil, nil, 0, err
				}
				d.skipped = append(d.skipped, entry)
				continue
			}
			ops = append(ops, linkOp(get(src, "h"), get(spec, "source_slot"), get(tgt, "h"), get(spec, "target_slot")))
			targets = append(targets, get(tgt, "h"))
		}
	}
	return ops, targets, restoredComps, nil
}

// inputEntry is _input_entry: a link-driven input the rollback did not restore, with its
// recorded value.
func inputEntry(comp string, spec *pyjson.Object, reason string) (*pyjson.Object, error) {
	target, err := ord.JoinOrd(comp, get(spec, "target_path").(string))
	if err != nil {
		return nil, err
	}
	return obj("target", target, "target_slot", get(spec, "target_slot"), "source_slot", get(spec, "source_slot"),
		"reason", reason, "value", get(spec, "prior")), nil
}
