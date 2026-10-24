package journal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// fixtureDir is the state dir the Python server wrote (tests/fixtures/journal/generate.py).
var fixtureDir = filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "journal", "python-written")

// fixtureBatchIDs are the batch ids of the Python-written journal, in file order.
func fixtureBatchIDs(t *testing.T) []any {
	views, err := New(fixtureDir).Views()
	if err != nil || len(views) != 6 {
		t.Fatalf("the Python-written journal: %d views, %v", len(views), err)
	}
	var out []any
	for _, v := range views {
		out = append(out, get(v, "batch_id"))
	}
	return out
}

// The batch ids the Python server journaled (tests/fixtures/journal/generate.py).
const (
	pyCreate     = "46aaed85dca34bc2ad345108c991b481"
	pyRollback   = "ce96d633fab8445dba"
	pyInDoubt    = "8ffa0b5f"
	pyLinkInDoub = "3a943320"
)

// fixtureID is the full id of the fixture batch whose id starts with prefix.
func fixtureID(t *testing.T, prefix string) string {
	t.Helper()
	for _, id := range fixtureBatchIDs(t) {
		if strings.HasPrefix(id.(string), prefix) {
			return id.(string)
		}
	}
	t.Fatalf("no fixture batch %s", prefix)
	return ""
}

func toolText(t *testing.T, err error) string {
	t.Helper()
	te, ok := err.(*jsonrpc.ToolError)
	if !ok {
		t.Fatalf("%#v is not a tool error", err)
	}
	return te.Msg
}

func TestAPythonInDoubtBatchIsRefused(t *testing.T) {
	j := New(fixtureDir)
	for _, prefix := range []string{pyInDoubt, pyLinkInDoub} {
		id := fixtureID(t, prefix)
		_, err := j.RollbackTarget(id)
		text := toolText(t, err)
		if !strings.HasPrefix(text, "batch "+id+" is in-doubt (the station may or may not have applied it); rolling "+
			"it back blindly could do harm, decide by hand. Journaled intent: {\"inverse_plan\":") ||
			!strings.Contains(text, `"station_name":"FakeStation"`) {
			t.Fatalf("%s: %s", prefix, text)
		}
	}
}

func TestAPythonRollbackIsNotRolledBackAgain(t *testing.T) {
	j := New(fixtureDir)
	id := fixtureID(t, pyRollback)
	_, err := j.RollbackTarget(id)
	if got := toolText(t, err); got != "batch "+id+" is itself a rollback of "+pyCreate+": roll forward by repeating the original write instead" {
		t.Fatal(got)
	}
	view, err := j.RollbackTarget(pyCreate)
	if err != nil || get(view, "state") != "completed" || get(view, "verdict") != "verified" {
		t.Fatalf("%v %v", view, err)
	}
	if got := toolText(t, j.EarlierRollbacks(pyCreate)); got != "batch "+pyCreate+" was already rolled back by "+id {
		t.Fatal(got)
	}
}

func TestTheBatchIdIsCheckedFirst(t *testing.T) {
	j := New(fixtureDir)
	for _, bad := range []any{nil, 5, "", strings.ToUpper(pyCreate), pyCreate + "0", pyCreate + "\n", pyCreate + "\n\n", " " + pyCreate} {
		if _, err := j.RollbackTarget(bad); toolText(t, err) != "batch_id must be a 32-character hex batch id" {
			t.Errorf("%v: %v", bad, err)
		}
	}
}

// unreadableJournal is a journal of batches idA (completed) and idB with extra lines.
func unreadableJournal(t *testing.T, extra string) *Journal {
	dir := stateDir(t)
	os.MkdirAll(dir, 0o700)
	body := `{"batch_id": "` + idA + `", "phase": "intent", "station_name": "S"}` + "\n" +
		`{"batch_id": "` + idA + `", "phase": "result", "verdict": "verified"}` + "\n" + extra +
		`{"batch_id": "` + idB + `", "phase": "intent", "station_name": "S"}` + "\n" +
		`{"batch_id": "` + idB + `", "phase": "result", "verdict": "verified"}` + "\n"
	os.WriteFile(filepath.Join(dir, "journal.jsonl"), []byte(body), 0o600)
	return New(dir)
}

// The journal check of test_tools_destructive.TestRollback.test_a_journal_line_whose_batch_is_unknown_refuses_every_rollback (that tool test runs under Go once n4_rollback is ported).
func TestAJournalLineWhoseBatchIsUnknownRefusesEveryRollback(t *testing.T) {
	j := unreadableJournal(t, "{\"batch_id\": \"\xff\"}\n{\"batch_id\": [1], \"phase\": \"intent\", \"rollback_of\": \""+idA+"\"}\n")
	for _, id := range []string{idA, idB, idC} {
		_, err := j.RollbackTarget(id)
		if got := toolText(t, err); !strings.HasPrefix(got, safety.ReasonJournalUnreadable+": batch "+id+
			" may be recorded on journal.jsonl line(s) 3 (UnicodeDecodeError)") {
			t.Errorf("%s: %s", id, got)
		}
	}
	_, err := j.RollbackTarget(idA)
	if got := toolText(t, err); !strings.Contains(got, "line(s) 3 (UnicodeDecodeError), 4 (TypeError); ") {
		t.Fatal(got)
	}
}

// The journal check of test_tools_destructive.TestRollback.test_an_unreadable_line_of_another_batch_does_not_block_a_rollback (that tool test runs under Go once n4_rollback is ported).
func TestAnUnreadableLineOfAnotherBatchDoesNotBlockARollback(t *testing.T) {
	j := unreadableJournal(t, `{"batch_id": [1], "phase": "intent", "rollback_of": "`+idB+`"}`+"\n"+
		`{"batch_id": "`+idC+`", "phase": "component-intent", "ops": 7}`+"\n")
	if view, err := j.RollbackTarget(idA); err != nil || get(view, "state") != "completed" {
		t.Fatalf("%v %v", view, err)
	}
	for id, line := range map[string]string{idB: "3", idC: "4"} {
		_, err := j.RollbackTarget(id)
		if got := toolText(t, err); !strings.HasPrefix(got, safety.ReasonJournalUnreadable+": batch "+id+
			" may be recorded on journal.jsonl line(s) "+line+" (TypeError); ") {
			t.Errorf("%s: %s", id, got)
		}
	}
}

func TestPythonWrittenBatchesListNewestFirst(t *testing.T) {
	views, err := New(fixtureDir).Views()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Batches(views)
	if err != nil || len(rows) != 6 {
		t.Fatalf("%d rows, %v", len(rows), err)
	}
	var got []string
	for _, r := range rows {
		o := r.(*pyjson.Object)
		id, _ := o.Get("batch_id")
		state, _ := o.Get("state")
		by, _ := o.Get("rolled_back_by")
		got = append(got, id.(string)[:8]+" "+state.(string)+" "+pyjson.Repr(by))
	}
	if !strings.HasPrefix(got[0], pyLinkInDoub+" in-doubt") || !strings.HasPrefix(got[1], pyInDoubt+" in-doubt") ||
		got[5] != pyCreate[:8]+" completed ['"+fixtureID(t, pyRollback)+"']" {
		t.Fatalf("%q", got)
	}
}
