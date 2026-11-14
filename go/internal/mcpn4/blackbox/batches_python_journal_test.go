package blackbox

// n4_list_batches on a state dir the Python server wrote (port of
// mcp-n4-kit/tests/test_list_batches.py; the fixture is mcp-n4-kit/tests/fixtures/journal/
// python-written, read in place and copied). The reply must be the one the Python server's
// tools_read.n4_list_batches computed in-process on the same files, byte for byte, frozen in
// mcp-n4-kit/tests/fixtures/oracles/list_batches.json (S17).

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// batchesCase is TestListBatchesOnAPythonJournal's fixture: a copy of the Python-written state
// dir, a read-only server on it (no session opened) and the test's frozen recording.
type batchesCase struct {
	t       *testing.T
	journal string
	srv     *Server
	fz      *frozen
}

func newBatchesCase(t *testing.T, test string) *batchesCase {
	state := filepath.Join(t.TempDir(), "state")
	copyTree(t, pythonWrittenDir(), state)
	r := NewRig(t, func(_ *Rig, o *Options) { o.StateDir = state })
	return &batchesCase{t: t, journal: filepath.Join(state, "journal.jsonl"), srv: r.Srv,
		fz: newFrozen(t, "list_batches", "TestListBatchesOnAPythonJournal."+test, [2]string{state, "<state>"})}
}

func (c *batchesCase) append(data string) {
	c.t.Helper()
	f, err := os.OpenFile(c.journal, os.O_WRONLY|os.O_APPEND, 0)
	if err == nil {
		_, err = f.WriteString(data)
		f.Close()
	}
	if err != nil {
		c.t.Fatal(err)
	}
}

// insert puts data after line `after` of the journal (1-based), between two batches.
func (c *batchesCase) insert(after int, data string) {
	c.t.Helper()
	raw, err := os.ReadFile(c.journal)
	if err != nil {
		c.t.Fatal(err)
	}
	lines := bytes.SplitAfter(raw, []byte("\n")) // the fixture's lines end in \n only
	out := append(bytes.Join(lines[:after], nil), data...)
	out = append(out, bytes.Join(lines[after:], nil)...)
	if err := os.WriteFile(c.journal, out, 0o600); err != nil {
		c.t.Fatal(err)
	}
}

// sameText fails unless the reply text to args is json.dumps of the recorded answer; it returns
// the raw result (a reply may hold NaN, which only json.loads reads).
func (c *batchesCase) sameText(args Args) *pyjson.Object {
	c.t.Helper()
	want := c.fz.wantText(args)
	got := callRaw(c.t, c.srv, "n4_list_batches", args)
	if isErr, _ := got.Get("isError"); isErr == true || resultText(got) != want {
		c.t.Errorf("args %v: text differs from Python's:\n got %.300s\nwant %.300s", args, resultText(got), want)
	}
	return got
}

// assertListed is the shared check of the unreadable-line tests.
func (c *batchesCase) assertListed(unreadable []any) map[string]any {
	c.t.Helper()
	for _, args := range []Args{{}, {"limit": 1}} {
		c.sameText(args)
	}
	reply := c.srv.OK("n4_list_batches", nil)
	if reply["total"] != 6.0 || !reflect.DeepEqual(reply["unreadable"], unreadable) {
		c.t.Errorf("total/unreadable = %v %v", reply["total"], reply["unreadable"])
	}
	if s, _ := reply["incomplete"].(string); !strings.Contains(s, "incomplete") {
		c.t.Errorf("incomplete = %v", reply["incomplete"])
	}
	return reply
}

func TestBatchesThePythonWrittenBatchesAreListedAsPythonListsThem(t *testing.T) {
	c := newBatchesCase(t, "test_the_python_written_batches_are_listed_as_python_lists_them")
	for _, args := range []Args{{}, {"limit": 1}, {"limit": 6}, {"limit": 1000}} {
		got := c.srv.Call("n4_list_batches", args)
		if want := c.fz.wantText(args); !reflect.DeepEqual(any(got.Structured), decoded(t, want)) {
			t.Errorf("args %v: structuredContent differs from Python's: %.300v", args, got.Structured)
		}
		if want := c.fz.wantText(args); got.Text != want {
			t.Errorf("args %v: text differs from Python's:\n got %.300s\nwant %.300s", args, got.Text, want)
		}
	}
	batches := objects(c.srv.OK("n4_list_batches", nil)["batches"])
	inDoubt, rolledBack := 0, 0
	for _, b := range batches {
		if b["state"] == "in-doubt" {
			inDoubt++
		}
		if rb, _ := b["rolled_back_by"].([]any); len(rb) > 0 { // a list of batch ids

			rolledBack++
		}
	}
	if inDoubt != 2 || rolledBack != 2 {
		t.Errorf("in-doubt %d, rolled back %d; want 2 and 2", inDoubt, rolledBack)
	}
}

func TestBatchesTornAndOddLinesAreReadAsPythonReadsThem(t *testing.T) {
	c := newBatchesCase(t, "test_torn_and_odd_lines_are_read_as_python_reads_them")
	c.append("{torn\n[1]\r{\"batch_id\": 1, \"phase\": \"intent\"}\r\n" +
		"{\"batch_id\": true, \"phase\": \"result\", \"verdict\": \"v\", \"ts\": NaN}\n")
	sc, _ := c.sameText(Args{}).Get("structuredContent")
	content, ok := sc.(*pyjson.Object)
	if !ok {
		t.Fatalf("no structuredContent: %v", sc)
	}
	if total, _ := content.Get("total"); dumps(t, total) != "7" {
		t.Errorf("total = %v", total)
	}
}

func TestBatchesALinePythonCannotDecodeIsOneUnreadableRowNotAFailedList(t *testing.T) {
	c := newBatchesCase(t, "test_a_line_python_cannot_decode_is_one_unreadable_row_not_a_failed_list")
	c.insert(4, "{\"batch_id\": \"\xff\", \"phase\": \"result\"}\n") // issue #448
	c.assertListed([]any{"unreadable journal line 5: UnicodeDecodeError"})
}

func TestBatchesAnUnhashableBatchIdIsOneUnreadableRowNotAFailedList(t *testing.T) {
	c := newBatchesCase(t, "test_an_unhashable_batch_id_is_one_unreadable_row_not_a_failed_list")
	c.insert(2, "{\"batch_id\": [1], \"phase\": \"intent\"}\n")
	c.append("{\"batch_id\": \"ee\", \"phase\": \"component-intent\", \"ops\": 7}\n" +
		"{\"batch_id\": \"ef\", \"phase\": \"result\", \"rollback_of\": {\"a\": 1}}\n")
	c.assertListed([]any{"unreadable journal line 3: TypeError",
		"unreadable journal line 16: TypeError",
		"unreadable journal line 17: TypeError"})
}

func TestBatchesAReadableJournalSaysNothingAboutUnreadableLines(t *testing.T) {
	c := newBatchesCase(t, "test_a_readable_journal_says_nothing_about_unreadable_lines")
	reply := c.srv.OK("n4_list_batches", nil)
	keys := make([]string, 0, len(reply))
	for k := range reply {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, []string{"batches", "total"}) {
		t.Errorf("keys = %v", keys)
	}
}

func TestBatchesItNeedsNoSessionAndNoJournalListsNothing(t *testing.T) {
	c := newBatchesCase(t, "test_it_needs_no_session_and_no_journal_lists_nothing")
	if err := os.Remove(c.journal); err != nil {
		t.Fatal(err)
	}
	if got := c.srv.OK("n4_list_batches", nil); !reflect.DeepEqual(got, map[string]any{"total": 0.0, "batches": []any{}}) {
		t.Errorf("reply = %v", got)
	}
}
