package journal

import (
	"math"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Journal is the append-only write-ahead record: an "intent" before each station op and a
// "result" after; entries of one batch share batch_id. An intent without a result, or a result
// flagged in_doubt, leaves the batch in-doubt: the station state is unknown.
type Journal struct{ File }

// New is Journal(state_dir): <state_dir>/journal.jsonl.
func New(stateDir string) *Journal { return &Journal{newFile(stateDir, "journal.jsonl")} }

// typeError is the TypeError Python raises; the server names only its type.
func typeError() error { return &pyjson.Exception{Type: "TypeError"} }

// dictKey is the identity of v as a key of a Python dict: equal keys collide as hash and ==
// make them (True is 1 is 1.0, False is 0 is -0.0), every NaN json.loads returns is the same
// object, and a list or dict cannot be a key (TypeError).
func dictKey(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "None", nil
	case bool:
		if x {
			return "i1", nil
		}
		return "i0", nil
	case string:
		return "s" + x, nil
	case pyjson.Number:
		if !x.IsFloat() {
			return "i" + x.String(), nil
		}
		f, _ := x.Float64()
		switch {
		case math.IsNaN(f):
			return "nan", nil
		case math.IsInf(f, 0) || f != math.Trunc(f):
			return "f" + pyjson.FloatRepr(f), nil
		}
		n, _ := big.NewFloat(f).Int(nil)
		return "i" + n.String(), nil
	}
	return "", typeError()
}

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

// merge is _merge: the intent without its phase, inverse/accepted/verdict reset, the relink and
// component ops when their write-ahead record exists, the result over it (without phase and
// ts) with result_ts, and the state.
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
		if !pyjson.Truthy(get(result, "in_doubt")) {
			state = "completed"
		}
	}
	view.Set("state", state)
	return view
}

// views is _views: the merged view of every batch with an intent, keyed as Python keys them,
// in the order of the batches' first intents.
func (j *Journal) views() ([]string, map[string]*pyjson.Object, error) {
	entries, err := j.Entries()
	if err != nil {
		return nil, nil, err
	}
	var order []string
	intents, results := map[string]*pyjson.Object{}, map[string]*pyjson.Object{}
	relinks, comps := map[string]any{}, map[string][]any{}
	for _, e := range entries {
		phase, _ := get(e, "phase").(string)
		if phase != "intent" && phase != "relink-intent" && phase != "component-intent" && phase != "result" {
			continue
		}
		key, err := dictKey(get(e, "batch_id"))
		if err != nil {
			return nil, nil, err
		}
		switch phase {
		case "intent":
			if _, seen := intents[key]; !seen {
				intents[key] = e
				order = append(order, key)
			}
		case "relink-intent":
			if _, seen := relinks[key]; !seen {
				relinks[key] = orEmpty(get(e, "ops"))
			}
		case "component-intent":
			if comps[key], err = extend(comps[key], orEmpty(get(e, "ops"))); err != nil {
				return nil, nil, err
			}
		case "result":
			results[key] = e
		}
	}
	out := map[string]*pyjson.Object{}
	for _, key := range order {
		r, hasRelinks := relinks[key]
		c, hasComps := comps[key]
		out[key] = merge(intents[key], results[key], r, hasRelinks, c, hasComps)
	}
	return order, out, nil
}

// Views is views(): the merged views of every journaled batch, in journal order.
func (j *Journal) Views() ([]*pyjson.Object, error) {
	order, byKey, err := j.views()
	if err != nil {
		return nil, err
	}
	out := make([]*pyjson.Object, len(order))
	for i, k := range order {
		out[i] = byKey[k]
	}
	return out, nil
}

// Read is read(batch_id): the merged view of one batch, nil when it was never journaled.
func (j *Journal) Read(batchID string) (*pyjson.Object, error) {
	_, byKey, err := j.views()
	if err != nil {
		return nil, err
	}
	return byKey["s"+batchID], nil
}

// RollbacksOf is rollbacks_of(batch_id): the views of every batch whose rollback_of is it.
func (j *Journal) RollbacksOf(batchID string) ([]*pyjson.Object, error) {
	views, err := j.Views()
	if err != nil {
		return nil, err
	}
	var out []*pyjson.Object
	for _, v := range views {
		if s, ok := get(v, "rollback_of").(string); ok && s == batchID {
			out = append(out, v)
		}
	}
	return out, nil
}
