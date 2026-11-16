package journal

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// typeError is the TypeError Python raises; the server names only its type.
func typeError() error { return &pyjson.Exception{Type: "TypeError"} }

// extend is list.extend(ops): a list adds its items, a string its characters, a dict its keys;
// a number or bool is not iterable (TypeError).
func extend(dst []any, ops any) ([]any, error) {
	switch x := ops.(type) {
	case []any:
		return append(dst, x...), nil
	case string:
		for i := 0; i < len(x); {
			_, n := pyjson.DecodeRune(x[i:])
			dst = append(dst, x[i:i+n])
			i += n
		}
		return dst, nil
	case *pyjson.Object:
		for _, k := range x.Keys() {
			dst = append(dst, k)
		}
		return dst, nil
	}
	return nil, typeError()
}

func get(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

// orEmpty is `entry.get("ops") or []`.
func orEmpty(v any) any {
	if pyjson.Truthy(v) {
		return v
	}
	return []any{}
}

// StateRefused is the state of a batch the station explicitly refused with nothing applied (its
// result record carries refused: true; added after the Python port).
const StateRefused = "refused"

// merge is _merge: the intent without its phase, inverse/accepted/verdict reset, the relink and
// component ops when their write-ahead record exists, the result over it (without phase and
// ts) with result_ts, and the state: in-doubt, completed or refused.
func merge(intent, result *pyjson.Object, relinks any, hasRelinks bool, comps []any, hasComps bool) *pyjson.Object {
	view := pyjson.NewObject()
	for _, k := range intent.Keys() {
		if k != "phase" {
			view.Set(k, get(intent, k))
		}
	}
	for _, k := range []string{"inverse", "accepted", "verdict"} {
		view.Set(k, nil)
	}
	if hasRelinks {
		view.Set("relink_ops", relinks)
	}
	if hasComps {
		view.Set("component_ops", comps)
	}
	state := "in-doubt"
	if result != nil {
		for _, k := range result.Keys() {
			if k != "phase" && k != "ts" {
				view.Set(k, get(result, k))
			}
		}
		view.Set("result_ts", get(result, "ts"))
		switch {
		case pyjson.Truthy(get(result, "refused")): // the station refused it, nothing applied
			state = StateRefused
		case !pyjson.Truthy(get(result, "in_doubt")):
			state = "completed"
		}
	}
	view.Set("state", state)
	return view
}
