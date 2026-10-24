package journal

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
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
	for _, bad := range []any{nil, 5, "", strings.ToUpper(pyCreate), pyCreate + "0", pyCreate + "\n\n", " " + pyCreate} {
		if _, err := j.RollbackTarget(bad); toolText(t, err) != "batch_id must be a 32-character hex batch id" {
			t.Errorf("%v: %v", bad, err)
		}
	}
	// re.match's "$" also matches before a final newline: the id passes, and is not journaled.
	if _, err := j.RollbackTarget(pyCreate + "\n"); toolText(t, err) != "unknown batch "+pyCreate+"\n: it is not in the journal" {
		t.Fatal(err)
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
