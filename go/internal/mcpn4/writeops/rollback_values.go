package writeops

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// written is one value a batch wrote: whole is false for invokeAction set, which writes the
// fallback's value only.
type written struct {
	value any
	whole bool
}

// writtenValues is _written_values: {(handle, slot): what the batch wrote}, from its ops.
func writtenValues(view *pyjson.Object) (map[string]written, error) {
	ops, err := items(orEmpty(get(view, "ops")))
	if err != nil {
		return nil, err
	}
	out := map[string]written{}
	for _, e := range ops {
		op, ok := e.(*pyjson.Object)
		if !ok {
			continue
		}
		if b, ok := get(op, "b").(*pyjson.Object); ok && get(op, "nm") == "s" {
			if err := remember(out, b, get(op, "h"), get(op, "n"), true); err != nil {
				return nil, err
			}
		}
		if get(op, "ssc") != "invokeAction" {
			continue
		}
		if arg, ok := get(op, "arg").(*pyjson.Object); ok && get(arg, "a") == "set" {
			if b, ok := get(arg, "b").(*pyjson.Object); ok {
				if err := remember(out, b, get(arg, "h"), "fallback", false); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

// remember is out[(h, n)] = (_bson_value(b), whole): the value first, then the key's hash.
func remember(out map[string]written, b *pyjson.Object, h, n any, whole bool) error {
	v, err := bsonValue(b)
	if err != nil {
		return err
	}
	k, err := pair(h, n)
	if err != nil {
		return err
	}
	out[k] = written{v, whole}
	return nil
}

// bsonValue is _bson_value: a BSON body's value in the form _coerce returns.
func bsonValue(b *pyjson.Object) (any, error) {
	tv, err := need(b, "t")
	if err != nil {
		return nil, err
	}
	t, ok := tv.(string)
	if !ok {
		return nil, &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(tv) + "' object has no attribute 'startswith'"}
	}
	if strings.HasPrefix(t, "baja:Status") {
		kids, err := items(sOf(b))
		if err != nil {
			return nil, err
		}
		values := map[string]any{}
		for _, k := range kids {
			c, err := dict(k, "get")
			if err != nil {
				return nil, err
			}
			key, err := pyjson.DictKey(get(c, "n"))
			if err != nil {
				return nil, unhashable(get(c, "n"))
			}
			values[key] = get(c, "v")
		}
		raw := values["svalue"]
		var value any = box.PyEqual(raw, "true")
		if strings.HasSuffix(t, "Numeric") {
			if value, err = plan.Float(orZero(raw)); err != nil {
				return nil, err
			}
		}
		status, ok := values["sstatus"]
		if !ok {
			status = "0"
		}
		return obj("value", value, "status", status), nil
	}
	v := get(b, "v")
	switch t {
	case "baja:Double":
		return plan.Float(orZero(v))
	case "baja:Boolean":
		return box.PyEqual(v, "true"), nil
	}
	if !pyjson.Truthy(v) {
		return "", nil
	}
	return v, nil
}

// sOf is body.get("s", []).
func sOf(b *pyjson.Object) any {
	if v, ok := b.Get("s"); ok {
		return v
	}
	return []any{}
}

func orZero(v any) any {
	if pyjson.Truthy(v) {
		return v
	}
	return "0.0"
}

// ordOf is _ord_of: the first recorded ORD whose handle == h ("" when none is).
func ordOf(targets *pyjson.Object, h any) string {
	for _, o := range targets.Keys() {
		if box.PyEqual(get(targets, o), h) {
			return o
		}
	}
	return ""
}

// unchangedSince is _unchanged_since: refuse to restore a slot that no longer holds what the
// batch wrote (audit F6); a later change is the operator's to undo, by hand.
func unchangedSince(c plan.Client, view *pyjson.Object, ops []any, targets *pyjson.Object) error {
	wrote, err := writtenValues(view)
	if err != nil {
		return err
	}
	for _, e := range ops {
		op := e.(*pyjson.Object)
		if get(op, "nm") != "s" {
			continue
		}
		h, err := need(op, "h")
		if err != nil {
			return err
		}
		n, err := need(op, "n")
		if err != nil {
			return err
		}
		k, err := pair(h, n)
		if err != nil {
			return err
		}
		w, ok := wrote[k]
		if !ok {
			continue
		}
		where := ordOf(targets, h)
		slot, ok := n.(string)
		if !ok {
			return &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(n) + "' object has no attribute 'split'"}
		}
		nodes, err := c.LoadTree(where, slotDepth(slot), box.LoadOptions{})
		if err != nil {
			return err
		}
		now, err := observeOp(nodes, op, slot)
		if err != nil {
			return err
		}
		seen := now
		if o, ok := now.(*pyjson.Object); ok && !w.whole {
			seen = get(o, "value")
		}
		if !box.PyEqual(seen, w.value) {
			return plan.ToolError("slot " + pyjson.Repr(n) + " on " + where + " changed since batch " +
				pyjson.Str(get(view, "batch_id")) + " wrote it (the batch wrote " + canonical(w.value) +
				", the station now holds " + canonical(seen) + "): refusing to overwrite a later change. " +
				"Restore it by hand with n4_set_slot if that is still wanted")
		}
	}
	return nil
}

// observeOp is `_observe(nodes, op["n"], op["b"]["t"])` where a KeyError or ValueError (the slot
// is gone, or no longer a Status) means it changed: nil.
func observeOp(nodes, op *pyjson.Object, slot string) (any, error) {
	b, err := need(op, "b")
	if err != nil {
		return nil, nil
	}
	body, ok := b.(*pyjson.Object)
	if !ok {
		return nil, notSubscriptable(b)
	}
	tv, err := need(body, "t")
	if err != nil {
		return nil, nil
	}
	t, ok := tv.(string)
	if !ok {
		return nil, &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(tv) + "' object has no attribute 'startswith'"}
	}
	now, err := plan.Observe(nodes, slot, t)
	var pe *plan.PyError
	var ve *pyjson.ValueError
	if errors.As(err, &ve) || errors.As(err, &pe) && pe.Type == "KeyError" {
		return nil, nil
	}
	return now, err
}

// flatten is _flatten: a nested snapshot body as its own slots and one spec per descendant
// component, parents first (breadth-first), with paths relative to the top, never handles.
func flatten(top int, body any) (*pyjson.Object, []any, error) {
	var specs []any
	type item struct {
		path string
		node any
	}
	queue := []item{{"", body}}
	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]
		node, err := dict(it.node, "get")
		if err != nil {
			return nil, nil, err
		}
		kids, err := items(sOf(node))
		if err != nil {
			return nil, nil, err
		}
		for _, k := range kids {
			kid, err := dict(k, "get")
			if err != nil {
				return nil, nil, err
			}
			if !box.IsComponentType(get(kid, "t")) {
				continue
			}
			name, err := plan.Name("component name", get(kid, "n"))
			if err != nil {
				return nil, nil, err
			}
			own, err := ownSlots(kid)
			if err != nil {
				return nil, nil, err
			}
			specs = append(specs, obj("top", top, "parent_path", it.path, "n", name, "b", own))
			path := name
			if it.path != "" {
				path = it.path + "/" + name
			}
			queue = append(queue, item{path, kid})
		}
	}
	b, err := dict(body, "items")
	if err != nil {
		return nil, nil, err
	}
	own, err := ownSlots(b)
	return own, specs, err
}

// ownSlots is _own_slots: a body without its component children (plain slots, wsAnnotation).
func ownSlots(body *pyjson.Object) (*pyjson.Object, error) {
	out := obj()
	for _, k := range body.Keys() {
		if k != "s" && k != "n" {
			out.Set(k, get(body, k))
		}
	}
	kids, err := items(sOf(body))
	if err != nil {
		return nil, err
	}
	var kept []any
	for _, k := range kids {
		c, err := dict(k, "get")
		if err != nil {
			return nil, err
		}
		if !box.IsComponentType(get(c, "t")) {
			kept = append(kept, c)
		}
	}
	if len(kept) > 0 {
		out.Set("s", kept)
	}
	return out, nil
}

func canonical(v any) string {
	b, _ := pyjson.Marshal(v, pyjson.Canonical) // decoded and computed values always encode
	return string(b)
}

// notSubscriptable is the TypeError of v["key"] on a value that is not a dict.
func notSubscriptable(v any) error {
	msg := "'" + plan.TypeName(v) + "' object is not subscriptable"
	switch v.(type) {
	case []any:
		msg = "list indices must be integers or slices, not str"
	case string:
		msg = "string indices must be integers, not 'str'"
	}
	return &plan.PyError{Type: "TypeError", Msg: msg}
}
