package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// applyInverse removes each copy under the name its add reply assigned ("nn"), else the planned
// one; the sets need no inverse (they go with the copy). replies hold the adds first.
func applyInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(*applyData)
	out := []any{}
	for i, r := range d.rows {
		nn := any(r.name)
		if i < len(replies) {
			if first, ok := replies[i].([]any); ok && len(first) > 0 {
				if reply, ok := first[0].(*pyjson.Object); ok {
					if v, ok := reply.Get("nn"); ok {
						nn = v
					}
				}
			}
		}
		out = append(out, obj("nm", "v", "h", d.parent, "n", nn))
	}
	return out, nil
}

// applyReadback checks, per row, that the copy exists, that each of its bindings holds the planned
// slot ORD, that each config slot holds the value and that the binding's target (loaded by that
// ORD) holds it too; then that the golden's own bound targets still hold what they did at the plan.
func applyReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(*applyData)
	_, nodes, err := plan.Handle(c, d.parentOrd, 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested, rows, verdict := []any{}, []any{}, "verified"
	for i, r := range d.rows {
		inv, _ := inverse[i].(*pyjson.Object)
		nn, _ := get(inv, "n").(string)
		requested = append(requested, obj("name", nn, "configs", r.configs))
		row := obj("name", nn, "configs", obj(), "verdict", "mismatch")
		if box.Node(nodes, nn) == nil {
			row.Set("differences", []any{"(copy not found)"})
		} else if row, err = d.readRow(c, i, nn); err != nil {
			return nil, nil, nil, "", err
		}
		if get(row, "verdict") != "verified" {
			verdict = "mismatch"
		}
		rows = append(rows, row)
	}
	unchanged, err := d.goldenUnchanged(c)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if !unchanged {
		verdict = "mismatch"
	}
	return requested, replies, obj("rows", rows, "golden_unchanged", unchanged), verdict, nil
}

// readRow reads copy nn of row i: its bindings, its configs and the targets they reach.
func (d *applyData) readRow(c plan.Client, i int, nn string) (*pyjson.Object, error) {
	copyOrd, err := ord.ChildOrd(d.parentOrd, nn)
	if err != nil {
		return nil, err
	}
	tree, err := c.LoadTree(copyOrd, cfgInstanceDepth, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	want, err := d.rowTargets(d.rows[i].name)
	if err != nil {
		return nil, err
	}
	diffs, configs := []any{}, obj()
	for _, b := range d.golden.bindings {
		path := "templateConfig/" + b.name + "/targetOrd"
		if got := pyjson.Str(get(box.Node(tree, path), "v")); got != want[b.name] {
			diffs = append(diffs, path+" holds "+pyjson.StrRepr(got)+", planned "+pyjson.StrRepr(want[b.name]))
		}
	}
	for _, v := range d.values[i] {
		b := d.golden.bySlot[v.slot]
		got, err := plan.Observe(tree, "templateConfig/"+v.slot, v.valueType)
		if err != nil {
			return nil, err
		}
		onTarget, err := observeAt(c, want[b.name], b.slot, v.valueType)
		if err != nil {
			return nil, err
		}
		configs.Set(v.slot, obj("requested", v.norm, "observed", got, "target_observed", onTarget))
		if !box.PyEqual(got, v.norm) || !box.PyEqual(onTarget, v.norm) {
			diffs = append(diffs, "templateConfig/"+v.slot+" or its target does not hold the requested value")
		}
	}
	row := obj("name", nn, "configs", configs, "verdict", "verified")
	if len(diffs) > 0 {
		row.Set("verdict", "mismatch")
		row.Set("differences", diffs)
	}
	return row, nil
}

// goldenUnchanged is true when each target the golden's configs are bound to holds what it held at the plan.
func (d *applyData) goldenUnchanged(c plan.Client) (bool, error) {
	same := true
	for _, slot := range d.golden.slotNames() {
		was := d.before[slot]
		if was == nil {
			continue
		}
		b := d.golden.bySlot[slot]
		at, err := b.targetOrd(d.golden.ord)
		if err != nil {
			return false, err
		}
		now, err := observeAt(c, at, b.slot, was.valueType)
		if err != nil {
			return false, err
		}
		same = same && box.PyEqual(now, was.value)
	}
	return same, nil
}

// observeAt is the value of slot of the component at ordText, read as valueType.
func observeAt(c plan.Client, ordText, slot, valueType string) (any, error) {
	_, nodes, err := plan.Handle(c, ordText, slotDepth(slot))
	if err != nil {
		return nil, err
	}
	return plan.Observe(nodes, slot, valueType)
}
