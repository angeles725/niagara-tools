package readtools_test

// Go twin of test_tools_destructive.TestListBatches.test_it_is_read_only_and_works_without_a_session
// (go_server.go_twin("readtools", "internals")), plus the Python-written journal through the tool.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func TestItIsReadOnlyAndWorksWithoutASession(t *testing.T) {
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: filepath.Join(t.TempDir(), "none")})
	text, isErr, out := s.Call(t, "n4_list_batches")
	if isErr || text != `{"total": 0, "batches": []}` || out == nil {
		t.Fatalf("%v %s", isErr, text)
	}
	for _, tool := range registry.Default().Read {
		if tool.Name == "n4_list_batches" {
			ann, _ := tool.Descriptor.Get("annotations")
			if ro, _ := ann.(*pyjson.Object).Get("readOnlyHint"); ro != true || tool.NeedsSession {
				t.Fatalf("annotations %v, needs_session %v", ro, tool.NeedsSession)
			}
			return
		}
	}
	t.Fatal("n4_list_batches is not a read tool")
}

func TestThePythonWrittenJournalIsListedNewestFirst(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "journal", "python-written")
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: dir})
	text, isErr, out := s.Call(t, "n4_list_batches", "limit", pyjson.IntLiteral("2"))
	total, _ := out.Get("total")
	rows, _ := out.Get("batches")
	if isErr || pyjson.Repr(total) != "6" || len(rows.([]any)) != 2 || !strings.Contains(text, `"state": "in-doubt"`) {
		t.Fatalf("%v %s", isErr, text)
	}
}

// Issue #448: an unreadable line is one row of the reply, never a failed list.
func TestAnUnreadableJournalLineIsListedNotAnError(t *testing.T) {
	for body, want := range map[string]string{
		"{\"batch_id\": \"b\", \"phase\": \"intent\"}\n\xff\n":                 "line 2: UnicodeDecodeError",
		"{\"batch_id\": [1], \"phase\": \"intent\"}\n":                         "line 1: TypeError",
		"{\"batch_id\": \"r\", \"phase\": \"intent\", \"rollback_of\": [1]}\n": "line 1: TypeError",
	} {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "journal.jsonl"), []byte(body), 0o600)
		s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: dir})
		text, isErr, _ := s.Call(t, "n4_list_batches")
		if isErr || !strings.Contains(text, `"incomplete": "this list is incomplete: 1 journal line(s)`) ||
			!strings.HasSuffix(text, `"unreadable": ["unreadable journal `+want+`"]}`) {
			t.Errorf("%q: %v %s", body, isErr, text)
		}
	}
}
