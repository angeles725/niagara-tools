package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// childObjects is node's children ("s"); a child list or child that is not an object is a
// malformed station reply, refused naming the node (path below the source) that holds it.
func childObjects(node *pyjson.Object, path string) ([]*pyjson.Object, error) {
	raw, has := node.Get("s")
	if !has {
		return nil, nil
	}
	label := path
	if label == "" {
		label = "."
	}
	refuse := plan.ToolError("malformed station reply: a child of " + pyjson.StrRepr(label) + " is not an object")
	list, ok := raw.([]any)
	if !ok {
		return nil, refuse
	}
	out := make([]*pyjson.Object, 0, len(list))
	for _, k := range list {
		kid, ok := k.(*pyjson.Object)
		if !ok {
			return nil, refuse
		}
		out = append(out, kid)
	}
	return out, nil
}

// dropNameOnly is body without children left with no type, value, children or flags, at every
// level: such a child is not a value and the copy's type supplies those slots. A child kept for
// its flags alone travels as name plus "f".
func dropNameOnly(body *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k != "s" {
			out.Set(k, v)
			continue
		}
		kids, _ := v.([]any)
		kept := []any{}
		for _, c := range kids {
			if o, ok := c.(*pyjson.Object); ok {
				if kid := dropNameOnly(o); !nameOnly(kid) {
					kept = append(kept, kid)
				}
			}
		}
		if len(kept) > 0 {
			out.Set("s", kept)
		}
	}
	return out
}

// nameOnly is a loaded child with no type, value, children or flags: an action or a frozen slot at
// its default, listed by name only. A frozen slot at its default value that carries instance flags
// ("f") is configuration an integrator set, not a name-only child: the copy keeps it for its flags.
func nameOnly(kid *pyjson.Object) bool {
	_, flagged := kid.Get("f")
	return !flagged && get(kid, "t") == nil && get(kid, "v") == nil && get(kid, "s") == nil
}

// carryFlags sets on each child of body the instance flags ("f") of the loaded node it was taken
// from: the snapshot keeps type, name and value only, and an add op child's "f" sets the copy's
// flags (baja ValueDocDecoder: setFlags on a frozen slot, the flags of a dynamic one).
func carryFlags(loaded, body *pyjson.Object) {
	kids, _ := get(body, "s").([]any)
	for _, k := range kids {
		kid, ok := k.(*pyjson.Object)
		if !ok {
			continue
		}
		if src := kidNamed(loaded, name(kid)); src != nil {
			if f, ok := get(src, "f").(string); ok {
				kid.Set("f", f)
			}
			carryFlags(src, kid)
		}
	}
}

func joinPath(path, n string) string {
	if path == "" {
		return n
	}
	return path + "/" + n
}

func asObj(v any) *pyjson.Object { o, _ := v.(*pyjson.Object); return o }

// kidNamed is node's child named n; nil when absent.
func kidNamed(node *pyjson.Object, n string) *pyjson.Object {
	kids, _ := get(node, "s").([]any)
	for _, k := range kids {
		if o, ok := k.(*pyjson.Object); ok && name(o) == n {
			return o
		}
	}
	return nil
}
