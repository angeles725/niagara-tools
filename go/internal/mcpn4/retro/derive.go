package retro

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func get(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

// str is `x if isinstance(x, str) else ""`.
func str(v any) string {
	s, _ := v.(string)
	return s
}

func candidate(key, change, target string, evidence []string, priority string) *pyjson.Object {
	o := pyjson.NewObject()
	items := make([]any, len(evidence))
	for i, e := range evidence {
		items[i] = e
	}
	for _, kv := range [][2]any{{"key", key}, {"change", change}, {"target", target}, {"evidence", items},
		{"count", len(evidence)}, {"type", "new"}, {"priority", priority}} {
		o.Set(kv[0].(string), kv[1])
	}
	return o
}

func batchEvidence(v *pyjson.Object) string {
	return fmt.Sprintf("batch %s (%s, %s)", safe(get(v, "batch_id")), safe(get(v, "tool")), safe(get(v, "ts")))
}

// derive is _derive: the candidates of entries already restricted to the window; byBatch maps
// every batch of the unfiltered journal by its Python dict key.
func derive(audit, views, observations []*pyjson.Object, byBatch map[string]*pyjson.Object) ([]*pyjson.Object, error) {
	var out []*pyjson.Object
	var order []*refusal // the reasons of one class (and its first row's advice) share a candidate
	groups := map[string][]string{}
	for _, e := range audit {
		if get(e, "outcome") != "refused" {
			continue
		}
		if r := refusalRow(str(get(e, "reason"))); r != nil {
			if _, seen := groups[r.key]; !seen {
				order = append(order, r)
			}
			groups[r.key] = append(groups[r.key], fmt.Sprintf("audit %s %s refused", safe(get(e, "ts")), safe(get(e, "tool"))))
		}
	}
	for _, r := range order {
		out = append(out, candidate("refusal:"+r.key, r.change, r.target, groups[r.key], r.priority))
	}
	for _, v := range verdicts {
		var evidence []string
		for _, b := range views {
			if get(b, "verdict") == v.verdict {
				evidence = append(evidence, batchEvidence(b))
			}
		}
		if evidence != nil {
			out = append(out, candidate("verdict:"+v.verdict, v.change, v.target, evidence, v.priority))
		}
	}
	var doubt []string
	for _, b := range views {
		if get(b, "state") == "in-doubt" {
			doubt = append(doubt, batchEvidence(b))
		}
	}
	if doubt != nil {
		out = append(out, candidate("in_doubt", "Add a guided in-doubt recovery step: inspect the station, "+
			"then decide on retry or rollback", "METHODOLOGY.md · section 4 step 6", doubt, "HIGH"))
	}
	box, err := boxErrors(audit, byBatch)
	if err != nil {
		return nil, err
	}
	ops := make([]string, 0, len(box))
	for op := range box {
		ops = append(ops, op)
	}
	sort.Strings(ops)
	for _, op := range ops {
		out = append(out, candidate("box_error:"+op, "Document or handle the BOX error seen on op "+op+
			" (cause and safe retry rule)", "METHODOLOGY.md · section 3", box[op], "MEDIUM"))
	}
	var seen []string
	for _, o := range observations {
		if n := get(o, "count"); pyjson.Truthy(n) {
			seen = append(seen, fmt.Sprintf("audit-less read %s n4_find_dangling_outputs found %s", safe(get(o, "ts")), pyjson.Str(n)))
		}
	}
	if seen != nil {
		out = append(out, candidate("dangling", "Make the dangling-output check part of the plan (link or remove "+
			"the unused out slots it reports)", "METHODOLOGY.md · section 4 step 7", seen, "MEDIUM"))
	}
	return out, nil
}

// boxErrors is the error audit lines by op: the journaled ops of the batch a "station call
// failed" line names (the tool when it journaled none), the tool of a BoxError line.
func boxErrors(audit []*pyjson.Object, byBatch map[string]*pyjson.Object) (map[string][]string, error) {
	box := map[string][]string{}
	for _, e := range audit {
		if get(e, "outcome") != "error" {
			continue
		}
		reason := str(get(e, "reason"))
		key, err := pyjson.DictKey(get(e, "batch_id"))
		if err != nil {
			return nil, err
		}
		var ops []string
		if batch := byBatch[key]; batch != nil && strings.Contains(reason, "station call failed") {
			if ops, err = opNames(get(batch, "ops")); err != nil {
				return nil, err
			}
			if ops == nil {
				ops = []string{safe(get(e, "tool"))}
			}
		} else if strings.HasPrefix(reason, "BoxError") {
			ops = []string{safe(get(e, "tool"))}
		}
		line := fmt.Sprintf("audit %s %s error", safe(get(e, "ts")), safe(get(e, "tool")))
		if id := get(e, "batch_id"); pyjson.Truthy(id) {
			line += " batch " + safe(id)
		}
		for _, op := range ops {
			box[op] = append(box[op], line)
		}
	}
	return box, nil
}

// opNames is sorted({_safe(op.get("nm")) for op in ops or [] if isinstance(op, dict)}): a
// string or dict iterates to no dict, a truthy number or bool is not iterable (TypeError).
func opNames(ops any) ([]string, error) {
	var items []any
	switch x := ops.(type) {
	case []any:
		items = x
	case string, *pyjson.Object:
	default:
		if pyjson.Truthy(x) {
			return nil, &pyjson.Exception{Type: "TypeError"}
		}
	}
	set := map[string]bool{}
	for _, op := range items {
		if o, ok := op.(*pyjson.Object); ok {
			set[safe(get(o, "nm"))] = true
		}
	}
	var names []string
	for n := range set {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}
