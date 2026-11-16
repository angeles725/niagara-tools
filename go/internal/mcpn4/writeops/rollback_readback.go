package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// rollbackInverse is _rollback_inverse: remove every component the batch's adds created (by the
// name the station assigned); none unless every op was an add.
func rollbackInverse(p *plan.Planned, replies []any) ([]any, error) {
	out := []any{}
	for i, op := range p.Ops {
		if o := op.(*pyjson.Object); i < len(replies) && get(o, "nm") == "a" {
			out = append(out, obj("nm", "v", "h", get(o, "h"), "n", assignedName(o, replies[i])))
		}
	}
	if len(out) != len(p.Ops) {
		return []any{}, nil
	}
	return out, nil
}

// rollbackReadback is _rollback_readback: each op's effect, every nested component with its
// type and annotation, and the configuration the rollback could not bring back (frozen
// children, inputs whose link was not restored), which caps the verdict at partial.
func rollbackReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(*rollbackData)
	ok, inputs := true, []any{}
	var relinks *pyjson.Object
	for i, e := range p.Ops {
		op := e.(*pyjson.Object)
		k, _ := pyjson.DictKey(get(op, "h")) // a recorded target's handle: hashable
		where, nm, depth := d.ordOf[k], get(op, "nm"), 2
		if nm == "s" {
			slot, isStr := get(op, "n").(string)
			if !isStr {
				return fail(&plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(get(op, "n")) + "' object has no attribute 'split'"})
			}
			depth = slotDepth(slot)
		}
		nodes, err := c.LoadTree(where, depth, box.LoadOptions{})
		if err != nil {
			return fail(err)
		}
		var good bool
		switch nm {
		case "v":
			in, err := contains(nodes, get(op, "n"))
			if err != nil {
				return fail(err)
			}
			good = !in
		case "a":
			nn := get(op, "n")
			if len(inverse) == len(p.Ops) {
				nn = get(inverse[i].(*pyjson.Object), "n")
			}
			if good, err = typed(nodes, nn, get(op, "b").(*pyjson.Object)); err != nil {
				return fail(err)
			}
			if good, err = andOutputsHeld(c, good, where, nn, get(op, "b").(*pyjson.Object)); err != nil {
				return fail(err)
			}
			if len(d.relinks) > 0 { // the runs always report; an ambiguous relink is also skipped
				relinks = d.report
				good = good && get(relinks, "skipped") == 0
			}
			comp, err := childOrd(where, nn)
			if err != nil {
				return fail(err)
			}
			for _, s := range d.unlinked {
				entry, err := inputEntry(comp, s.(*pyjson.Object), "link source outside the removed subtree")
				if err != nil {
					return fail(err)
				}
				inputs = append(inputs, entry)
			}
		case "f":
			if good, err = flagsRestored(c, where, nodes, op); err != nil {
				return fail(err)
			}
		default:
			if good, err = restoredOrDefault(c, nodes, op); err != nil {
				return fail(err)
			}
		}
		ok = ok && good
	}
	// every nested component is really there, with the recorded type and annotation; the two
	// lists are paired by position, so a length difference is itself a mismatch
	created := d.created
	ok = ok && len(d.components) == len(created)
	for i, s := range d.components {
		if i >= len(created) || created[i] == nil { // a frozen child that already existed
			continue
		}
		head, leaf := rpartition(created[i].(string))
		nodes, err := c.LoadTree(head, 2, box.LoadOptions{})
		if err != nil {
			return fail(err)
		}
		body := get(s.(*pyjson.Object), "b").(*pyjson.Object)
		n := box.Node(nodes, leaf)
		ok = ok && n != nil && box.PyEqual(get(n, "t"), get(body, "t")) &&
			box.PyEqual(get(box.Node(nodes, leaf+"/wsAnnotation"), "v"), annotation(body))
		if ok, err = andOutputsHeld(c, ok, head, leaf, body); err != nil {
			return fail(err)
		}
	}
	var frozen []any
	unread := false
	for _, check := range d.frozenChecks {
		gap, err := frozenCheck(c, check.(*pyjson.Object))
		if err != nil {
			return fail(err)
		}
		if gap != nil {
			frozen = append(frozen, gap)
			_, e := gap.Get("readback_error")
			unread = unread || e
		}
	}
	inputs = append(append([]any{}, d.skipped...), inputs...)
	observed := obj("restored", ok)
	if relinks != nil {
		observed.Set("relinks", relinks)
	}
	if len(frozen) > 0 {
		observed.Set("frozen_config_not_restored", frozen)
	}
	if len(inputs) > 0 {
		observed.Set("link_inputs_not_restored", inputs)
	}
	verdict := "verified"
	switch { // an unread frozen child is unobserved, not lost: below mismatch only
	case !ok:
		verdict = "mismatch"
	case unread:
		verdict = "unverified"
	case len(frozen) > 0 || len(inputs) > 0:
		verdict = "partial"
	}
	return obj("rolled_back", d.rollbackOf), replies, observed, verdict, nil
}

func fail(err error) (any, any, any, string, error) { return nil, nil, nil, "", err }

// contains is `name in nodes`: a node path is a string, an unhashable name is TypeError.
func contains(nodes *pyjson.Object, name any) (bool, error) {
	if _, err := pyjson.DictKey(name); err != nil {
		return false, unhashable(name)
	}
	s, ok := name.(string)
	return ok && box.Node(nodes, s) != nil, nil
}

// typed is `nn in nodes and nodes[nn].get("t") == body["t"]`.
func typed(nodes *pyjson.Object, nn any, body *pyjson.Object) (bool, error) {
	in, err := contains(nodes, nn)
	if err != nil || !in {
		return false, err
	}
	t, err := need(body, "t")
	return err == nil && box.PyEqual(get(box.Node(nodes, nn.(string)), "t"), t), err
}

// restored is `op["n"] in nodes and _observe(nodes, op["n"], op["b"]["t"]) == _bson_value(op["b"])`.
func restored(nodes, op *pyjson.Object) (bool, error) {
	n, err := need(op, "n")
	if err != nil {
		return false, err
	}
	if in, err := contains(nodes, n); err != nil || !in {
		return false, err
	}
	b, err := need(op, "b")
	if err != nil {
		return false, err
	}
	body, isDict := b.(*pyjson.Object)
	if !isDict {
		return false, notSubscriptable(b)
	}
	tv, err := need(body, "t")
	if err != nil {
		return false, err
	}
	t, isStr := tv.(string)
	if !isStr {
		return false, &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(tv) + "' object has no attribute 'startswith'"}
	}
	now, err := plan.Observe(nodes, n.(string), t)
	if err != nil {
		return false, err
	}
	want, err := bsonValue(body)
	return err == nil && box.PyEqual(now, want), err
}

// annotation is _annotation: the wsAnnotation value of a component body, or None.
func annotation(body *pyjson.Object) any {
	kids, _ := get(body, "s").([]any) // _own_slots kept a list of dicts
	for _, k := range kids {
		if c := k.(*pyjson.Object); get(c, "n") == "wsAnnotation" {
			return get(c, "v")
		}
	}
	return nil
}

func rpartition(path string) (string, string) {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return "", path
	}
	return path[:i], path[i+1:]
}
