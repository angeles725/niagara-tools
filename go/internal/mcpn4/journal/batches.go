package journal

import (
	"fmt"
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ListBatches is n4_list_batches' reply: the total and the newest limit batches; when a line is
// unreadable (issue #448), a sentence saying the list is incomplete and one row per such line.
func (j *Journal) ListBatches(limit int) (*pyjson.Object, error) {
	views, bad, err := j.Scan()
	if err != nil {
		return nil, err
	}
	rows, err := Batches(views)
	if err != nil {
		return nil, err
	}
	out := pyjson.NewObject()
	out.Set("total", len(rows))
	out.Set("batches", rows[:min(max(limit, 0), len(rows))])
	if len(bad) > 0 {
		out.Set("incomplete", fmt.Sprintf("this list is incomplete: %d journal line(s) are unreadable; a batch "+
			"recorded on one is missing or partial here, and n4_rollback refuses every batch such a line may "+
			"belong to", len(bad)))
		lines := make([]any, len(bad))
		for i, u := range bad {
			lines[i] = fmt.Sprintf("unreadable journal line %d: %s", u.Line, u.Reason)
		}
		out.Set("unreadable", lines)
	}
	return out, nil
}

// Batches is the batch list of n4_list_batches (tools_read.py): one summary per view, newest
// first, with the ids of the batches recorded as rolling it back.
func Batches(views []*pyjson.Object) ([]any, error) {
	rolled := map[string][]any{}
	for _, v := range views {
		if of := get(v, "rollback_of"); pyjson.Truthy(of) {
			key, err := pyjson.DictKey(of)
			if err != nil {
				return nil, err
			}
			rolled[key] = append(rolled[key], get(v, "batch_id"))
		}
	}
	out := make([]any, 0, len(views))
	for i := len(views) - 1; i >= 0; i-- {
		v := views[i]
		key, err := pyjson.DictKey(get(v, "batch_id"))
		if err != nil {
			return nil, err
		}
		by := rolled[key]
		if by == nil {
			by = []any{}
		}
		row := pyjson.NewObject()
		for _, k := range []string{"batch_id", "tool", "station_name", "ts", "result_ts", "state", "verdict", "rollback_of"} {
			row.Set(k, get(v, k))
		}
		row.Set("rolled_back_by", by)
		out = append(out, row)
	}
	return out, nil
}

// batchIDPattern is _BATCH_ID under re.match: "$" also matches before a final newline.
var batchIDPattern = regexp.MustCompile(`^[0-9a-f]{32}\n?$`)

func toolError(msg string) error { return &jsonrpc.ToolError{Msg: msg} }

func canonical(v any) string {
	b, _ := pyjson.Marshal(v, pyjson.Canonical) // decoded JSON values always encode
	return string(b)
}

// RollbackTarget makes the journal checks _rollback_plan makes before the station one: a batch
// id, journaled, not itself a rollback and not in-doubt (the station may or may not have applied
// it). It returns the batch's view.
func (j *Journal) RollbackTarget(batchID any) (*pyjson.Object, error) {
	bid, ok := batchID.(string)
	if !ok || !batchIDPattern.MatchString(bid) {
		return nil, toolError("batch_id must be a 32-character hex batch id")
	}
	view, err := j.Read(bid)
	if err != nil {
		return nil, err
	}
	if view == nil {
		return nil, toolError("unknown batch " + bid + ": it is not in the journal")
	}
	if of := get(view, "rollback_of"); pyjson.Truthy(of) {
		return nil, toolError("batch " + bid + " is itself a rollback of " + pyjson.Str(of) +
			": roll forward by repeating the original write instead")
	}
	if get(view, "state") == "in-doubt" {
		intent := pyjson.NewObject()
		for _, k := range []string{"tool", "ops", "inverse_plan", "station_name", "targets"} {
			intent.Set(k, get(view, k))
		}
		return nil, toolError("batch " + bid + " is in-doubt (the station may or may not have applied it); " +
			"rolling it back blindly could do harm, decide by hand. Journaled intent: " + canonical(intent))
	}
	return view, nil
}

// EarlierRollbacks makes the check _rollback_plan makes after the station one: no earlier
// rollback of batchID is in-doubt or done, and none failed after the station accepted its ops.
func (j *Journal) EarlierRollbacks(batchID string) error {
	earlier, err := j.RollbacksOf(batchID)
	if err != nil {
		return err
	}
	for _, e := range earlier {
		id, ok := e.Get("batch_id")
		if !ok {
			return &pyjson.Exception{Type: "KeyError"}
		}
		switch {
		case get(e, "state") == "in-doubt":
			return toolError("a rollback of batch " + batchID + " (" + pyjson.Str(id) + ") is in-doubt: inspect the station")
		case get(e, "verdict") != "failed":
			return toolError("batch " + batchID + " was already rolled back by " + pyjson.Str(id))
		case pyjson.Truthy(get(e, "readback_failed")):
			return toolError("a rollback of batch " + batchID + " (" + pyjson.Str(id) + ") is in-doubt: the station " +
				"accepted its ops but the read-back failed, so it may already be applied; inspect the station, do " +
				"not retry blindly. Accepted: " + canonical(get(e, "accepted")))
		}
	}
	return nil
}
