package plan

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// PlanKeys are the Planned.Keys a token authorizes: a truthy one joins the hashed plan, in
// this order (_process).
var PlanKeys = []string{"relinks", "unlinked_inputs", "components", "outgoing_links_broken", "snapshot_truncated"}

// IntentKeys are the Planned.Keys a truthy value of which the intent records.
var IntentKeys = []string{"rollback_of", "relinks", "components"}

// priorKinds are _PRIOR_KINDS: inverse entries whose spec may carry `prior`, the runtime value
// a link drives into an input.
var priorKinds = []string{"relink", "unlinked_input"}

// priorKind is _prior_kind: the key under which entry carries a spec with `prior`, else "".
func priorKind(entry any) (string, *pyjson.Object) {
	e, ok := entry.(*pyjson.Object)
	if !ok {
		return "", nil
	}
	for _, k := range priorKinds {
		if spec, ok := get(e, k).(*pyjson.Object); ok {
			if _, has := spec.Get("prior"); has {
				return k, spec
			}
		}
	}
	return "", nil
}

// HashedInverse is _hashed_inverse: the inverse a token authorizes, every `prior` dropped. A
// prior changes whenever its link source does, so hashing it would make a dry run and its
// confirming re-plan disagree by timing alone; the journal keeps the full inverse.
func HashedInverse(inverse []any) []any {
	out := make([]any, 0, len(inverse))
	for _, entry := range inverse {
		kind, spec := priorKind(entry)
		if kind == "" {
			out = append(out, entry)
			continue
		}
		e := copyObj(entry.(*pyjson.Object))
		bare := pyjson.NewObject()
		for _, k := range spec.Keys() {
			if k != "prior" {
				bare.Set(k, get(spec, k))
			}
		}
		e.Set(kind, bare)
		out = append(out, e)
	}
	return out
}

// LinkInputValues is _link_input_values: the `prior` values an inverse carries, the dry run's
// preview (never hashed). A spec without target_path or target_slot is Python's KeyError.
func LinkInputValues(inverse []any) ([]any, error) {
	out := []any{}
	for _, entry := range inverse {
		_, spec := priorKind(entry)
		if spec == nil {
			continue
		}
		v := obj()
		for _, k := range [][2]string{{"target_path", "target_path"}, {"target_slot", "target_slot"}, {"value", "prior"}} {
			x, ok := spec.Get(k[1])
			if !ok {
				return nil, &PyError{Type: "KeyError", Msg: pyjson.StrRepr(k[1])}
			}
			v.Set(k[0], x)
		}
		out = append(out, v)
	}
	return out, nil
}

// withKeys sets the truthy keys of p.Keys on o, in the order of keys.
func withKeys(o *pyjson.Object, p *Planned, keys []string) {
	if p.Keys == nil {
		return
	}
	for _, k := range keys {
		if v := get(p.Keys, k); pyjson.Truthy(v) {
			o.Set(k, v)
		}
	}
}

func copyObj(o *pyjson.Object) *pyjson.Object {
	c := pyjson.NewObject()
	for _, k := range o.Keys() {
		c.Set(k, get(o, k))
	}
	return c
}
