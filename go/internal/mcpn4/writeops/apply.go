package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Apply is n4_apply (#934): a golden deployed template instance under parent_ord copied once per
// row of a table, each copy's ConfigBindings re-pointed at the copy's own components and the row's
// config slots set on the copy's templateConfig, in one batch under one confirmation token. The
// copy is n4_copy_component's (same body, strips and licence check). The sets are explicit ops
// after the adds: BTemplateConfig.changed propagates a config to its binding's target only outside
// Context.decoding, and an add decodes its body. The inverse removes every copy.
var Apply = plan.Impl{ScopeOrds: copyScope, Plan: applyPlan, Inverse: applyInverse, Readback: applyReadback}

// applyData is what the read-back needs: the parent, the rows with the values each sets, the golden
// and the golden's bound values before the batch.
type applyData struct {
	parent    any
	parentOrd string
	rows      []applyRow
	values    [][]*applyValue
	golden    *applyGolden
	before    map[string]*applyBefore // by config slot
}

// applyBefore is the value the golden's target of a config slot held at the plan, and its type.
type applyBefore struct {
	valueType string
	value     any
}

func applyPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	rows, err := applyRows(args)
	if err != nil {
		return nil, err
	}
	goldenArg, _ := args.Get("golden")
	golden, err := plan.ExistingName("golden", goldenArg)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	parentH, parentNodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	if box.Node(parentNodes, golden) == nil {
		return nil, plan.ToolError("component " + pyjson.StrRepr(golden) + " not found under " + parentOrd)
	}
	names, configured := []string{}, map[string]bool{}
	for _, r := range rows {
		if box.Node(parentNodes, r.name) != nil {
			return nil, plan.ToolError("a child named " + pyjson.StrRepr(r.name) + " already exists under " + parentOrd)
		}
		names = append(names, r.name)
		for _, k := range r.configs.Keys() {
			configured[k] = true
		}
	}
	src, err := loadCopySource(c, parentOrd, golden, parentNodes)
	if err != nil {
		return nil, err
	}
	goldenOrd, err := ord.ChildOrd(parentOrd, golden)
	if err != nil {
		return nil, err
	}
	g, err := readGolden(goldenOrd, golden, src.root, configured)
	if err != nil {
		return nil, err
	}
	bodies, dupNotes, _, err := copyAddresses(src.ct, src.root, src.body, golden, names, obj())
	if err != nil {
		return nil, err
	}
	licenceNotes, err := withinLicence(c, "this copy", func(a capacityAdd) error {
		return a.addTree(kinds{}, src.ct, src.root, len(rows))
	})
	if err != nil {
		return nil, err
	}
	d := &applyData{parent: parentH, parentOrd: parentOrd, rows: rows, golden: g, before: map[string]*applyBefore{}}
	ops, sets, inverse := []any{}, []any{}, []any{}
	for i, r := range rows {
		targets, err := d.rowTargets(r.name)
		if err != nil {
			return nil, err
		}
		body, err := repoint(bodies[i], targets)
		if err != nil {
			return nil, err
		}
		ops = append(ops, obj("nm", "a", "h", parentH, "n", r.name, "b", body))
		inverse = append(inverse, obj("nm", "v", "h", parentH, "n", r.name))
		vals, err := g.values(r)
		if err != nil {
			return nil, err
		}
		d.values = append(d.values, vals)
		for _, v := range vals {
			sets = append(sets, setOp(parentH, r.name+"/templateConfig/"+v.slot, v.body))
			was, err := g.goldenValue(g.bySlot[v.slot], v.valueType)
			if err != nil {
				return nil, err
			}
			d.before[v.slot] = &applyBefore{v.valueType, was}
		}
	}
	ops = append(ops, sets...)
	notes := append(strippedNotes(src.stripped), applyNotes...)
	notes = append(notes, src.linksNote()...)
	notes = append(notes, dupNotes...)
	notes = append(notes, fallbackNotes(golden, src.body)...)
	notes = append(notes, licenceNotes...)
	targets := pyjson.NewObject()
	targets.Set(parentOrd, parentH)
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets, Data: d, Writes: len(ops)}, nil
}

// applyNotes are the notes every plan of n4_apply carries.
var applyNotes = []any{
	"Each copy's ConfigBindings are re-pointed from the golden's handle to the slot ORD of the copy's own " +
		"component (station:|slot:...): a later rename or move of a copy breaks them; rename the copy's " +
		"components, not the copy.",
	"The configs are set by explicit ops after the adds: the station propagates a templateConfig slot to its " +
		"binding's target only outside the decoding of an add.",
}

// rowTargets is the slot ORD each binding of the copy named name must hold: its own component.
func (d *applyData) rowTargets(name string) (map[string]string, error) {
	out := map[string]string{}
	for _, b := range d.golden.bindings {
		copyOrd, err := ord.ChildOrd(d.parentOrd, name)
		if err != nil {
			return nil, err
		}
		at, err := b.targetOrd(copyOrd)
		if err != nil {
			return nil, err
		}
		out[b.name] = at
	}
	return out, nil
}
