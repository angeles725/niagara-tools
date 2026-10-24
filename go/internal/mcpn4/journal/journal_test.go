package journal

// Go twins of test_safety.TestJournal (read side) and TestJournalMergeRules
// (go_server.go_twin("journal", "unit")).

import (
	"os"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func dumps(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func read(t *testing.T, j *Journal, id string) string {
	t.Helper()
	v, err := j.Read(id)
	if err != nil {
		t.Fatal(err)
	}
	if v == nil {
		return "None"
	}
	return dumps(t, v)
}

func TestReadMergesIntentAndResultAndFlagsInDoubt(t *testing.T) {
	j := New(stateDir(t))
	if read(t, j, "b1") != "None" {
		t.Fatal("a view before any file")
	}
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "intent", "tool": "t", "ops": [1], "inverse_plan": [2], "station_name": "S", "ts": "T0"}`)
	if got := read(t, j, "b1"); got != `{"batch_id": "b1", "tool": "t", "ops": [1], "inverse_plan": [2], "station_name": "S", "ts": "T0", `+
		`"inverse": null, "accepted": null, "verdict": null, "state": "in-doubt"}` {
		t.Fatalf("intent only: %s", got)
	}
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "result", "accepted": "ok", "inverse": [3], "verdict": "verified", "ts": "T1", "tool": "x"}`)
	if got := read(t, j, "b1"); got != `{"batch_id": "b1", "tool": "x", "ops": [1], "inverse_plan": [2], "station_name": "S", "ts": "T0", `+
		`"inverse": [3], "accepted": "ok", "verdict": "verified", "result_ts": "T1", "state": "completed"}` {
		t.Fatalf("with result: %s", got)
	}
	if read(t, j, "nope") != "None" {
		t.Fatal("unknown batch")
	}
}

func TestReadIgnoresTornLinesAndFlagsInDoubtResults(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "intent", "tool": "t"}`)
	fh, _ := os.OpenFile(j.Path, os.O_WRONLY|os.O_APPEND, 0)
	fh.WriteString("{torn\n[1]\n\"s\"\n\n")
	fh.Close()
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "result", "inverse": [], "verdict": "unverified", "in_doubt": true}`)
	v, _ := j.Read("b1")
	if s, _ := v.Get("state"); s != "in-doubt" {
		t.Fatalf("state %v", s)
	}
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "result", "in_doubt": 0}`) // the last result wins
	v, _ = j.Read("b1")
	if s, _ := v.Get("state"); s != "completed" {
		t.Fatalf("state %v", s)
	}
}

func TestRollbacksOfListsBatchesThatRollBackAnother(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "r1", "phase": "intent", "rollback_of": "b1"}`,
		`{"batch_id": "r1", "phase": "result", "verdict": "failed", "inverse": []}`,
		`{"batch_id": "x", "phase": "intent", "rollback_of": ["b1"]}`,
		`{"batch_id": "r2", "phase": "intent", "rollback_of": "b1"}`)
	got, err := j.RollbacksOf("b1")
	if err != nil || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	for i, want := range [][2]string{{"r1", "completed"}, {"r2", "in-doubt"}} {
		if id, _ := got[i].Get("batch_id"); id != want[0] {
			t.Fatalf("%d: %v", i, id)
		}
		if s, _ := got[i].Get("state"); s != want[1] {
			t.Fatalf("%d: %v", i, s)
		}
	}
}

func view(t *testing.T, texts ...string) *pyjson.Object {
	t.Helper()
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "intent", "tool": "t"}`)
	for _, text := range texts {
		e := entry(t, text)
		e.Set("batch_id", "b1")
		if err := j.Append(e); err != nil {
			t.Fatal(err)
		}
	}
	v, err := j.Read("b1")
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRelinkAndComponentOpsArePresentExactlyWhenARecordExists(t *testing.T) {
	v := view(t)
	for _, k := range []string{"relink_ops", "component_ops"} {
		if _, ok := v.Get(k); ok {
			t.Fatalf("%s present without a record", k)
		}
	}
}

func TestAnEmptyRecordIsPresentForBothKinds(t *testing.T) {
	v := view(t, `{"phase": "relink-intent"}`, `{"phase": "component-intent", "ops": []}`)
	if got := dumps(t, v); got != `{"batch_id": "b1", "tool": "t", "inverse": null, "accepted": null, "verdict": null, `+
		`"relink_ops": [], "component_ops": [], "state": "in-doubt"}` {
		t.Fatal(got)
	}
}

func TestRecordedOpsAreKeptForBothKinds(t *testing.T) {
	v := view(t, `{"phase": "relink-intent", "ops": [1]}`, `{"phase": "relink-intent", "ops": [9]}`,
		`{"phase": "component-intent", "ops": [2]}`, `{"phase": "component-intent", "ops": "ab"}`,
		`{"phase": "component-intent", "ops": {"k": 1}}`, `{"phase": "component-intent", "ops": 0}`)
	r, _ := v.Get("relink_ops")
	c, _ := v.Get("component_ops")
	if dumps(t, r) != "[1]" || dumps(t, c) != `[2, "a", "b", "k"]` {
		t.Fatalf("%s %s", dumps(t, r), dumps(t, c))
	}
}

// TestBatchIdsAreKeyedAsPythonDictKeys pins the views safety.Journal gives (checked on CPython
// 3.15): 1, true and 1.0 are one key, every NaN is one key, a result's batch_id replaces the
// intent's, and an unhashable batch_id is a TypeError only in a phase that keys it.
func TestBatchIdsAreKeyedAsPythonDictKeys(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": 1, "phase": "intent", "n": "int"}`, `{"batch_id": true, "phase": "intent", "n": "bool"}`,
		`{"batch_id": 1.0, "phase": "result", "verdict": "float"}`, `{"batch_id": NaN, "phase": "intent"}`,
		`{"batch_id": NaN, "phase": "result"}`, `{"batch_id": 0.5, "phase": "intent"}`, `{"phase": "intent", "n": "none"}`,
		`{"batch_id": null, "phase": "result"}`, `{"batch_id": [1], "phase": "other"}`)
	views, err := j.Views()
	if err != nil {
		t.Fatal(err)
	}
	list := make([]any, len(views))
	for i, v := range views {
		list[i] = v
	}
	if got := dumps(t, list); got != `[{"batch_id": 1.0, "n": "int", "inverse": null, "accepted": null, "verdict": "float", "result_ts": null, "state": "completed"}, `+
		`{"batch_id": NaN, "inverse": null, "accepted": null, "verdict": null, "result_ts": null, "state": "completed"}, `+
		`{"batch_id": 0.5, "inverse": null, "accepted": null, "verdict": null, "state": "in-doubt"}, `+
		`{"n": "none", "inverse": null, "accepted": null, "verdict": null, "batch_id": null, "result_ts": null, "state": "completed"}]` {
		t.Fatal(got)
	}
	// A line whose batch id cannot key a batch (or whose ops cannot extend a list) is one
	// unreadable line of a known batch, not a failed journal (issue #448).
	for _, bad := range []string{`{"batch_id": [1], "phase": "intent"}`, `{"batch_id": {}, "phase": "result"}`,
		`{"batch_id": "b", "phase": "component-intent", "ops": 5}`, `{"batch_id": "b", "phase": "result", "rollback_of": [1]}`} {
		j := New(stateDir(t))
		appendAll(t, j.File, `{"batch_id": "a", "phase": "intent"}`, bad)
		views, bad2, err := j.Scan()
		if err != nil || len(views) != 1 || len(bad2) != 1 || bad2[0].Line != 2 || bad2[0].Reason != "TypeError" || !bad2[0].Known {
			t.Errorf("%s: %v %+v %v", bad, len(views), bad2, err)
		}
	}
}

func TestANonUtf8LineIsUnreadableNotAFailedJournal(t *testing.T) {
	j := New(stateDir(t))
	appendAll(t, j.File, `{"batch_id": "b1", "phase": "intent"}`)
	fh, _ := os.OpenFile(j.Path, os.O_WRONLY|os.O_APPEND, 0)
	fh.Write([]byte{0xff, '\r', '\n', '{'})
	fh.Close()
	appendAll(t, j.File, `{"batch_id": "b2", "phase": "intent"}`)
	views, bad, err := j.Scan()
	if err != nil || len(views) != 1 || len(bad) != 1 || bad[0] != (Unreadable{Line: 2, Reason: "UnicodeDecodeError"}) {
		t.Fatalf("%d views, %+v, %v", len(views), bad, err)
	}
}
