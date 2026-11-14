package blackbox

// The session retro draft as the server lists and answers it (port of
// mcp-n4-kit/tests/test_retro.py, TestServerTool and TestDanglingReadsReachTheDraft).

import (
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

const retroTool = "n4_session_retro_draft"

// bid is "%032x" % n, the batch ids B1..B5 of the Python tests.
func bid(n int) string { return fmt.Sprintf("%032x", n) }

// journalBatch is test_retro.journal_batch: an intent and, unless withResult is false, a result;
// nil ops are the default ({"nm": "s"},).
func journalBatch(id string, ts any, verdict any, inDoubt, withResult bool, ops []any) []any {
	if ops == nil {
		ops = []any{obj("nm", "s")}
	}
	rows := []any{obj("batch_id", id, "ts", ts, "tool", "n4_set_slot", "ops", ops, "phase", "intent",
		"station_name", "LLM", "inverse_plan", []any{}, "targets", obj())}
	if withResult {
		res := obj("batch_id", id, "ts", ts, "phase", "result", "accepted", nil, "inverse", []any{}, "verdict", verdict)
		if inDoubt {
			res.Set("in_doubt", true)
		}
		rows = append(rows, res)
	}
	return rows
}

// mismatchState is a state dir holding one mismatch batch of 2026-10-01, written before the
// server starts.
func mismatchState(t *testing.T) string {
	state := filepath.Join(t.TempDir(), "s")
	appendLines(t, state, "journal.jsonl", journalBatch(bid(1), "2026-10-01T11:00:00+00:00", "mismatch", false, true, nil)...)
	return state
}

func retroCandidates(s *Server, args Args) []any {
	return candidateKeys(s.OK(retroTool, args))
}

func TestRetroToolIsListedReadOnlyInBothModes(t *testing.T) {
	for _, allow := range []bool{false, true} {
		s := Start(t, Options{AllowWrites: allow, StateDir: filepath.Join(t.TempDir(), "s")})
		tool, ok := toolsByName(t, s)[retroTool]
		if !ok {
			t.Fatalf("allow-writes %v: %s is not listed", allow, retroTool)
		}
		if want := map[string]any{"readOnlyHint": true, "openWorldHint": false}; !reflect.DeepEqual(tool["annotations"], want) {
			t.Errorf("allow-writes %v: annotations = %v", allow, tool["annotations"])
		}
		s.Close()
	}
}

func TestRetroToolReturnsMarkdownAndCandidatesForTheServerStateDir(t *testing.T) {
	s := Start(t, Options{StateDir: mismatchState(t)})
	out := s.OK(retroTool, Args{"since": "2000-01-01T00:00:00Z"})
	if got := candidateKeys(out); !reflect.DeepEqual(got, []any{"verdict:mismatch"}) {
		t.Errorf("candidates = %v", got)
	}
	if md, _ := out["markdown"].(string); !strings.Contains(md, "## Proposed kit deltas") {
		t.Errorf("markdown = %.200s", md)
	}
	if got := retroCandidates(s, Args{"since": "2026-10-02T00:00:00Z"}); len(got) != 0 {
		t.Errorf("candidates since 2026-10-02 = %v", got)
	}
}

func TestRetroSinceDefaultsToTheServerStartSoADraftCoversTheSession(t *testing.T) {
	s := Start(t, Options{StateDir: mismatchState(t)}) // the batch was written before this start
	if got := retroCandidates(s, Args{}); len(got) != 0 {
		t.Errorf("default window candidates = %v", got)
	}
	if got := retroCandidates(s, Args{"since": ""}); len(got) != 0 { // empty is the default too
		t.Errorf("empty since candidates = %v", got)
	}
	if got := retroCandidates(s, Args{"since": "2000-01-01T00:00:00Z"}); len(got) != 1 {
		t.Errorf("whole window candidates = %v", got)
	}
}

func TestRetroBadSinceIsAToolError(t *testing.T) {
	s := Start(t, Options{StateDir: filepath.Join(t.TempDir(), "s")})
	if text := s.Err(retroTool, Args{"since": "nope"}); text != "not an ISO-8601 timestamp: 'nope'" {
		t.Errorf("refusal = %q", text)
	}
}

// A dangling report is an in-memory observation: only the server process knows it.
func TestRetroADanglingReportBecomesACandidateOfThisSession(t *testing.T) {
	r := stationRig(t)
	r.AddWithOut("A", 1)
	if got := objects(dangling(r, Args{"ord": Folder})["dangling"]); len(got) != 1 {
		t.Fatalf("dangling = %v", got)
	}
	dangling(r, Args{"ord": Folder})
	out := r.Srv.OK(retroTool, nil)
	if out["station"] != StationName {
		t.Errorf("station = %v", out["station"])
	}
	cands := objects(out["candidates"])
	if got := candidateKeys(out); !reflect.DeepEqual(got, []any{"dangling"}) {
		t.Fatalf("candidates = %v", got)
	}
	if cands[0]["count"] != 2.0 {
		t.Errorf("count = %v", cands[0]["count"])
	}
	item := regexp.MustCompile(`^audit-less read \d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\+00:00 n4_find_dangling_outputs found 1$`)
	evidence, _ := cands[0]["evidence"].([]any)
	if len(evidence) == 0 {
		t.Error("the candidate has no evidence")
	}
	for _, e := range evidence {
		if s, _ := e.(string); !item.MatchString(s) {
			t.Errorf("evidence item %q", e)
		}
	}
	if md, _ := out["markdown"].(string); !strings.Contains(md, fmt.Sprintf("# Retro — %s · %v · ", StationName, out["date"])) {
		t.Errorf("markdown = %.200s", md)
	}
}
