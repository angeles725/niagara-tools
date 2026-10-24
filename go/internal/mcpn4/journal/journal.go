package journal

import (
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

// Unreadable is one journal line the views leave out (issue #448): not UTF-8 (Reason
// "UnicodeDecodeError", its batch unknown), or a line of a batch phase whose batch_id or
// rollback_of cannot key a dict or whose component ops cannot extend a list ("TypeError", its
// batch known: BatchID and RollbackOf are the line's values).
type Unreadable struct {
	Line                int
	Reason              string
	Known               bool
	BatchID, RollbackOf any
}

// views is _scan: the merged view of every batch with an intent, keyed as Python keys them, in
// the order of the batches' first intents, and the unreadable lines.
func (j *Journal) views() ([]string, map[string]*pyjson.Object, []Unreadable) {
	var order []string
	var bad []Unreadable
	intents, results := map[string]*pyjson.Object{}, map[string]*pyjson.Object{}
	relinks, comps := map[string]any{}, map[string][]any{}
	for _, l := range j.lines() {
		e := l.entry
		if e == nil {
			bad = append(bad, Unreadable{Line: l.n, Reason: "UnicodeDecodeError"})
			continue
		}
		phase, _ := get(e, "phase").(string)
		if phase != "intent" && phase != "relink-intent" && phase != "component-intent" && phase != "result" {
			continue
		}
		key, err := pyjson.DictKey(get(e, "batch_id"))
		if of := get(e, "rollback_of"); err == nil && (phase == "intent" || phase == "result") && pyjson.Truthy(of) {
			_, err = pyjson.DictKey(of) // n4_list_batches keys rolled_back_by by it
		}
		var ops []any
		if err == nil && phase == "component-intent" {
			ops, err = extend(nil, orEmpty(get(e, "ops")))
		}
		if err != nil {
			bad = append(bad, Unreadable{Line: l.n, Reason: "TypeError", Known: true,
				BatchID: get(e, "batch_id"), RollbackOf: get(e, "rollback_of")})
			continue
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
			comps[key] = append(comps[key], ops...)
			if comps[key] == nil {
				comps[key] = []any{}
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
	return order, out, bad
}

// Scan is scan(): Views and the unreadable lines, from one read of the file.
func (j *Journal) Scan() ([]*pyjson.Object, []Unreadable, error) {
	order, byKey, bad := j.views()
	out := make([]*pyjson.Object, len(order))
	for i, k := range order {
		out[i] = byKey[k]
	}
	return out, bad, nil
}

// UnreadableFor is unreadable_for(batch_id), the rollback check of issue #448: the unreadable
// lines that may belong to batchID. That is every line whose batch is unknown (not UTF-8), and a
// line whose batch_id or rollback_of is batchID (a lost rollback of a batch could let it be
// rolled back twice). A line of another known batch blocks only that batch.
func (j *Journal) UnreadableFor(batchID string) []Unreadable {
	_, _, bad := j.views()
	var out []Unreadable
	for _, u := range bad {
		id, _ := u.BatchID.(string)
		of, _ := u.RollbackOf.(string)
		if !u.Known || u.BatchID != nil && id == batchID || u.RollbackOf != nil && of == batchID {
			out = append(out, u)
		}
	}
	return out
}

// Views is views(): the merged views of every journaled batch, in journal order.
func (j *Journal) Views() ([]*pyjson.Object, error) {
	views, _, err := j.Scan()
	return views, err
}

// Read is read(batch_id): the merged view of one batch, nil when it was never journaled.
func (j *Journal) Read(batchID string) (*pyjson.Object, error) {
	_, byKey, _ := j.views()
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
