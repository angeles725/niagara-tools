package blackbox

// The journal as the destructive tests craft and read it (port of state_files.Journal): append
// a record the server reads, and merge one batch the way the journal does, with no server code.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// appendJournal appends raw bytes to journal.jsonl, creating the state dir 0700 and the file
// 0600 as the server does.
func (d *dRig) appendJournal(data []byte) {
	d.T.Helper()
	if err := os.MkdirAll(d.StateDir, 0o700); err != nil {
		d.T.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(d.StateDir, "journal.jsonl"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		d.T.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		d.T.Fatal(err)
	}
}

// journalAppend appends entry as one JSON line (Journal.append).
func (d *dRig) journalAppend(entry map[string]any) {
	d.T.Helper()
	line, err := json.Marshal(entry)
	if err != nil {
		d.T.Fatal(err)
	}
	d.appendJournal(append(line, '\n'))
}

// journalEntries are the journal's JSON object lines; an undecodable line is skipped.
func (d *dRig) journalEntries() []map[string]any {
	data, _ := os.ReadFile(filepath.Join(d.StateDir, "journal.jsonl"))
	out := []map[string]any{}
	for _, raw := range bytes.Split(data, []byte("\n")) {
		var entry map[string]any
		if json.Unmarshal(raw, &entry) == nil && entry != nil {
			out = append(out, entry)
		}
	}
	return out
}

// views merges every batch: the first intent, the last result, the relink and component
// write-ahead ops, and state in-doubt when the result is missing or flagged in_doubt.
func (d *dRig) views() map[string]map[string]any {
	intents, results := map[string]map[string]any{}, map[string]map[string]any{}
	relinks, comps := map[string]any{}, map[string][]any{}
	var order []string
	for _, e := range d.journalEntries() {
		bid, ok := e["batch_id"].(string)
		if !ok {
			continue
		}
		switch e["phase"] {
		case "intent":
			if intents[bid] == nil {
				intents[bid] = e
				order = append(order, bid)
			}
		case "relink-intent":
			if _, seen := relinks[bid]; !seen {
				ops, _ := e["ops"].([]any)
				relinks[bid] = append([]any{}, ops...)
			}
		case "component-intent":
			ops, _ := e["ops"].([]any)
			comps[bid] = append(comps[bid], ops...)
		case "result":
			results[bid] = e
		}
	}
	views := map[string]map[string]any{}
	for _, bid := range order {
		view := map[string]any{"inverse": nil, "accepted": nil, "verdict": nil}
		for k, v := range intents[bid] {
			if k != "phase" {
				view[k] = v
			}
		}
		view["inverse"], view["accepted"], view["verdict"] = nil, nil, nil
		if ops, ok := relinks[bid]; ok {
			view["relink_ops"] = ops
		}
		if ops, ok := comps[bid]; ok {
			view["component_ops"] = ops
		}
		result := results[bid]
		for k, v := range result {
			if k != "phase" && k != "ts" {
				view[k] = v
			}
		}
		if result != nil {
			view["result_ts"] = result["ts"]
		}
		view["state"] = "completed"
		if result == nil || result["in_doubt"] == true {
			view["state"] = "in-doubt"
		}
		views[bid] = view
	}
	return views
}

// batch is Journal.read: the merged view of one batch, nil when it was never journaled.
func (d *dRig) batch(id any) map[string]any {
	s, _ := id.(string)
	return d.views()[s]
}

// rollbacksOf are the merged views of every batch recorded as rolling back id.
func (d *dRig) rollbacksOf(id any) []map[string]any {
	out := []map[string]any{}
	for _, v := range d.views() {
		if v["rollback_of"] == id {
			out = append(out, v)
		}
	}
	return out
}
