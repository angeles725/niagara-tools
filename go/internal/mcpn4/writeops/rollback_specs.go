package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// inverseParts splits a recorded inverse: its ops (dicts with "nm"), then the validated relink
// specs, then the validated unlinked inputs, each in the order Python's comprehensions check them.
func inverseParts(inverse []any) (ops, relinks, unlinked []any, err error) {
	for _, kind := range []string{"nm", "relink", "unlinked_input"} {
		for _, e := range inverse {
			o, ok := e.(*pyjson.Object)
			if !ok {
				continue
			}
			v, has := o.Get(kind)
			switch {
			case !has:
			case kind == "nm":
				ops = append(ops, o)
			case kind == "relink":
				spec, err := relinkSpec(v)
				if err != nil {
					return nil, nil, nil, err
				}
				relinks = append(relinks, spec)
			default:
				spec, err := unlinkedSpec(v)
				if err != nil {
					return nil, nil, nil, err
				}
				unlinked = append(unlinked, spec)
			}
		}
	}
	return ops, relinks, unlinked, nil
}

// recordedTargets is `view.get("targets") or {}`, refused when it is empty or misses the handle
// of an op.
func recordedTargets(view *pyjson.Object, ops []any, bid string) (*pyjson.Object, error) {
	var targets *pyjson.Object
	if v := get(view, "targets"); pyjson.Truthy(v) {
		t, err := dict(v, "values")
		if err != nil {
			return nil, err
		}
		targets = t
	}
	refused := targets == nil
	for _, op := range ops {
		if refused {
			break
		}
		refused = ordOf(targets, get(op.(*pyjson.Object), "h")) == ""
	}
	if refused {
		return nil, plan.ToolError("batch " + bid + " does not record which components its inverse touches: refusing")
	}
	return targets, nil
}

// relinkSpec is _relink_spec: a journaled relink, paths and slot names only, never handles.
func relinkSpec(v any) (*pyjson.Object, error) {
	spec, ok := v.(*pyjson.Object)
	keys := []string{"source_path", "source_slot", "target_path", "target_slot"}
	for _, k := range keys {
		if _, isStr := get(spec, k).(string); !ok || !isStr {
			return nil, plan.ToolError("batch records a malformed relink: refusing")
		}
	}
	out := obj()
	for _, k := range keys {
		s := get(spec, k).(string)
		if s != "" {
			for _, part := range strings.Split(s, "/") {
				if _, err := plan.Name(k, part); err != nil {
					return nil, err
				}
			}
		}
		out.Set(k, s)
	}
	if prior, ok := get(spec, "prior").(*pyjson.Object); ok { // journaled since v0.28.0, absent before
		out.Set("prior", prior)
	}
	return out, nil
}

// unlinkedSpec is _unlinked_spec: an input whose link came from outside the removed subtree.
func unlinkedSpec(v any) (*pyjson.Object, error) {
	spec, ok := v.(*pyjson.Object)
	if !ok {
		return nil, plan.ToolError("batch records a malformed unlinked input: refusing")
	}
	withSource := copyObj(spec)
	withSource.Set("source_path", "")
	out, err := relinkSpec(withSource)
	if err != nil {
		return nil, err
	}
	trimmed := obj()
	for _, k := range out.Keys()[1:] {
		trimmed.Set(k, get(out, k))
	}
	return trimmed, nil
}

func copyObj(o *pyjson.Object) *pyjson.Object {
	c := obj()
	for _, k := range o.Keys() {
		c.Set(k, get(o, k))
	}
	return c
}

func list(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
