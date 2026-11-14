package blackbox

// The retro draft is the Python server's draft (port of mcp-n4-kit/tests/test_retro.py,
// TestTheDraftIsPythonsDraft): the tool's reply, byte for byte, is the one the Python server's
// tools_read.n4_session_retro_draft computed in-process on the same state dir, frozen in
// mcp-n4-kit/tests/fixtures/oracles/retro_draft.json (S17).

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const (
	retroWhole  = "1970-01-01T00:00:00Z"
	retroCanary = "pw-Zk93-unique-canary"
)

// retroRefusals are test_retro.REFUSALS' reasons, worded as the raise sites word them.
var retroRefusals = []string{
	"confirmation_token is missing or malformed: run a dry run first",
	"confirmation_token does not match this tool, these arguments",
	"confirmation_token expired: run the dry run again",
	"confirmation_token already used: run the dry run again",
	"station identity not verified: reconnect with n4_connect",
	"ord 'station:|slot:/X' is outside the write scope",
	"write budget exhausted: 2 writes already executed in this session",
	"refusing to write 'out' on its own: Status slots are written whole",
	"action 'foo' is not allowed in this version (allowed: set)",
	"type must look like module:Type, got 'x'",
}

// everyRefusal are reasons of every refusal class (test_retro.EVERY_REFUSAL); tierRefusal is
// tiers_write_test.go's, which TestUnknownVersionIsRefusedAsTierC keeps true to the server.
var everyRefusal = append(append([]string{}, retroRefusals...),
	"write budget too small: the rollback needs 3 writes, 1 left",
	"not connected: call n4_connect first",
	"value_type must be one of: bool, numeric, string",
	"no --write-scope configured: every write is refused",
	"ord '/etc' is not a plain station ORD",
	tierRefusal("LLM"),
	"something no class knows",
)

// auditRow is test_retro.audit_row; args are already redacted (state_files.redact_args).
func auditRow(ts, tool any, outcome, reason string, batchID any, args *pyjson.Object) *pyjson.Object {
	if args == nil {
		args = obj()
	}
	return obj("ts", ts, "tool", tool, "batch_id", batchID, "dry_run", outcome == "planned",
		"outcome", outcome, "reason", reason, "args_redacted", args)
}

// draftCase is TestTheDraftIsPythonsDraft's fixture: a state dir (not created yet), a server on
// it and the test's frozen recording.
type draftCase struct {
	t   *testing.T
	dir string
	srv *Server
	fz  *frozen
}

func newDraftCase(t *testing.T, test string) *draftCase {
	dir := filepath.Join(t.TempDir(), "state")
	today := time.Now().UTC().Format("2006-01-02")
	return &draftCase{t: t, dir: dir, srv: Start(t, Options{StateDir: dir}),
		fz: newFrozen(t, "retro_draft", "TestTheDraftIsPythonsDraft."+test, [2]string{dir, "<state>"}, [2]string{today, oracleToday})}
}

func (c *draftCase) audit(rows ...any) { c.t.Helper(); appendLines(c.t, c.dir, "audit.jsonl", rows...) }
func (c *draftCase) journal(rows ...any) {
	c.t.Helper()
	appendLines(c.t, c.dir, "journal.jsonl", rows...)
}

func (c *draftCase) remove(name string) {
	c.t.Helper()
	if err := os.Remove(filepath.Join(c.dir, name)); err != nil {
		c.t.Fatal(err)
	}
}

// sameDraft is assertSameDraft: Python drafted these args, and the reply is its draft.
func (c *draftCase) sameDraft(args Args) map[string]any {
	c.t.Helper()
	answer, _ := c.fz.want(args).(*pyjson.Object)
	want, ok := answer.Get("draft")
	if !ok {
		c.t.Fatalf("Python refused %v", args)
	}
	got := c.srv.Call(retroTool, args)
	sameReply(c.t, got, dumps(c.t, want))
	return got.Structured
}

// sameError is assertSameError: Python refused these args, and the tool error is its text.
func (c *draftCase) sameError(args Args) {
	c.t.Helper()
	answer, _ := c.fz.want(args).(*pyjson.Object)
	raw, _ := answer.Get("error")
	text, ok := raw.(string)
	if !ok {
		c.t.Fatalf("Python drafted %v", args)
	}
	if got := c.srv.Call(retroTool, args); !got.IsError || got.Text != text {
		c.t.Errorf("reply %v, want the tool error %q", got.Raw, text)
	}
}

func TestRetroDraftEveryRefusalClassAndTheTierRefusal(t *testing.T) {
	c := newDraftCase(t, "test_every_refusal_class_and_the_tier_refusal")
	for i, reason := range append(append([]string{}, everyRefusal...), everyRefusal...) {
		c.audit(auditRow(fmt.Sprintf("2026-10-01T10:%02d:00+00:00", i), "n4_set_slot", "refused", reason, nil, nil))
	}
	out := c.sameDraft(Args{"since": retroWhole})
	if !containsAny(candidateKeys(out), "refusal:tier") {
		t.Errorf("candidates = %v", candidateKeys(out))
	}
	if !strings.Contains(out["markdown"].(string), "; +1 more") { // more than five evidence items
		t.Error("the markdown does not cap the evidence")
	}
}

func TestRetroDraftVerdictsInDoubtAndBoxErrors(t *testing.T) {
	c := newDraftCase(t, "test_verdicts_in_doubt_and_box_errors")
	for i, verdict := range []string{"mismatch", "failed", "partial", "unverified", "verified"} {
		c.journal(journalBatch(bid(i+1), "2026-10-01T11:00:00+00:00", verdict, false, true, nil)...)
	}
	c.journal(journalBatch("d1", "2026-10-01T11:01:00Z", nil, false, false,
		[]any{obj("nm", "z"), obj("nm", "a"), obj("nm", "a"), "odd", obj("nm", "bad name")})...)
	c.journal(journalBatch("d2", "2026-10-01T11:02:00", "unverified", true, true, nil)...)
	c.journal(journalBatch("d3", "2026-10-01T11:03:00+00:00", nil, false, false, []any{})...)
	failed := "station call failed after the intent was journaled"
	c.audit(auditRow("2026-10-01T11:01:01+00:00", "n4_create_component", "error", failed, "d1", nil),
		auditRow("2026-10-01T11:03:01+00:00", "n4_remove", "error", failed, "d3", nil),
		auditRow("2026-10-01T11:04:00+00:00", "n4_link", "error", failed, "never-journaled", nil),
		auditRow("2026-10-01T11:05:00+00:00", "n4_set_slot", "error", "BoxError: HTTP 500", nil, nil),
		auditRow("2026-10-01T11:06:00+00:00", "n4_set_slot", "error", "ValueError: no", nil, nil),
		auditRow("2026-10-01T11:07:00+00:00", "n4 bad|tool", "error", "BoxError: x", "", nil))
	c.sameDraft(Args{"since": retroWhole})
}

func TestRetroDraftOfThePythonWrittenStateDir(t *testing.T) {
	c := newDraftCase(t, "test_the_python_written_state_dir")
	copyTree(t, pythonWrittenDir(), c.dir)
	if out := c.sameDraft(Args{"since": retroWhole}); !containsAny(candidateKeys(out), "in_doubt") {
		t.Errorf("candidates = %v", candidateKeys(out))
	}
}

func TestRetroDraftNoFrictionAndNoStateDirGiveTheHonestyLine(t *testing.T) {
	c := newDraftCase(t, "test_no_friction_and_no_state_dir_give_the_honesty_line")
	c.sameDraft(Args{"since": retroWhole}) // the dir does not exist yet
	c.audit(auditRow("2026-10-01T10:00:00+00:00", "n4_set_slot", "executed", "verdict=verified", bid(1), nil))
	c.journal(journalBatch(bid(1), "2026-10-01T10:00:00+00:00", "verified", false, true, nil)...)
	if out := c.sameDraft(Args{"since": retroWhole}); len(objects(out["candidates"])) != 0 {
		t.Errorf("candidates = %v", out["candidates"])
	}
	def := c.srv.OK(retroTool, nil) // the window starts at the server start
	if len(objects(def["candidates"])) != 0 || !strings.Contains(def["markdown"].(string), "- Window: since 20") {
		t.Errorf("default draft = %v", def)
	}
}

func TestRetroDraftWindowParsesTimestampsAsFromisoformatDoes(t *testing.T) {
	c := newDraftCase(t, "test_the_window_parses_timestamps_as_fromisoformat_does")
	stamps := []any{"2026-10-01T10:00:00+00:00", "2026-10-01T10:00:00Z", "2026-10-01 10:00",
		"2026-10-01T09:30:00-01:00", "20261001T113000", "2026-W40-4T10:00",
		"2026W404T1000", "2026-10-01T10:00:00.5+00:00", "2026-10-01T24:00",
		" 2026-10-01T10:00:00 ", "2026-10-01T10:00:00+01", "2026-10-01T10:00Zjunk",
		"not a time", "", nil, 7, "2026-13-01T00:00", "2026-10-01é10:00", "2026-10-01T10:00:00,123456789"}
	for i, ts := range stamps {
		c.audit(with(auditRow(nil, "n4_set_slot", "refused", fmt.Sprintf("write budget exhausted: %d", i), nil, nil), "ts", ts))
	}
	for _, since := range []string{"2026-10-01T10:00:00Z", "2026-10-01T10:00", "2026-10-01T11:00:00+01:00",
		"2026-10-01T10:30Z", " 2026-W40-4 ", "2026-10-02", "20261001"} {
		c.sameDraft(Args{"since": since})
	}
}

func TestRetroDraftBadSinceValuesAreTheSameValueError(t *testing.T) {
	c := newDraftCase(t, "test_bad_since_values_are_the_same_value_error")
	for _, since := range []string{"yesterday", "2026-10-01T25:00", "2026-02-30", "2026-10-01T10:00+24:00",
		strings.Repeat("\u0000", 9), "2026-10-01T10:00:00Zé", "2026-10-01T10:61"} {
		c.sameError(Args{"since": since})
	}
}

func TestRetroDraftOddAuditValuesAreSummarizedAsPythonSummarizesThem(t *testing.T) {
	c := newDraftCase(t, "test_odd_audit_values_are_summarized_as_python_summarizes_them")
	for _, outcome := range []any{"refused", nil, 1, true, pyjson.FloatNumber(1.0), "a|b", "planned",
		strings.Repeat("x", 65), 2.5, false, 0, "None"} {
		c.audit(with(auditRow("2026-10-01T10:00:00+00:00", 5, "refused", "write budget exhausted", nil, nil), "outcome", outcome))
	}
	c.journal(obj("batch_id", "v", "ts", "2026-10-01T10:00:00+00:00", "tool", "t", "phase", "intent", "ops", []any{}),
		obj("batch_id", "v", "phase", "result", "verdict", 3))
	c.sameDraft(Args{"since": retroWhole})
}

func TestRetroDraftValuesPythonCannotHashOrIterateAreTheSameInternalError(t *testing.T) {
	c := newDraftCase(t, "test_values_python_cannot_hash_or_iterate_are_the_same_internal_error")
	c.audit(with(auditRow("2026-10-01T10:00:00+00:00", "n4_set_slot", "planned", "", nil, nil), "outcome", []any{"x"}))
	c.sameError(Args{"since": retroWhole})
	c.remove("audit.jsonl")
	c.journal(obj("batch_id", "n", "ts", "2026-10-01T10:00:00+00:00", "phase", "intent", "ops", 5))
	c.audit(auditRow("2026-10-01T10:00:01+00:00", "n4_set_slot", "error", "station call failed", "n", nil))
	c.sameError(Args{"since": retroWhole})
	c.remove("audit.jsonl")
	c.audit(with(auditRow("2026-10-01T10:00:02+00:00", "n4_set_slot", "error", "x", nil, nil), "batch_id", obj("a", 1)))
	c.sameError(Args{"since": retroWhole})
}

// Issue #466: a line that is not UTF-8 is left out and named; the rest is drafted. Bad lines end
// in CRLF and in CR, and one is cut at the 8192-byte read chunk boundary (a lead byte at 8191,
// an ASCII byte at 8192), where readlines() failed the file.
func TestRetroDraftAnUnreadableAuditLineCostsThatLineOnly(t *testing.T) {
	c := newDraftCase(t, "test_an_unreadable_audit_line_costs_that_line_only")
	row := func(i int) string {
		return dumps(t, auditRow(fmt.Sprintf("2026-10-01T10:%02d:00+00:00", i), "n4_set_slot", "refused",
			fmt.Sprintf("write budget exhausted: %d", i), nil, nil))
	}
	head := []byte(row(0) + "\r\n" + "\xff bad\r\n" + row(1) + "\r" + "\xfe\r")
	cut := []byte(`{"x": "` + strings.Repeat("a", 8191-len(head)-7) + "\xc3(" + "\"}\n")
	if n := len(head) + bytes.IndexByte(cut, 0xc3); n != 8191 {
		t.Fatalf("the lead byte is at %d, want 8191", n)
	}
	c.audit(auditRow(retroWhole, "t", "refused", "x", nil, nil)) // creates it private
	if err := os.WriteFile(filepath.Join(c.dir, "audit.jsonl"), append(append(head, cut...), row(2)+"\n"...), 0o600); err != nil {
		t.Fatal(err)
	}
	out := c.sameDraft(Args{"since": retroWhole})
	md := out["markdown"].(string)
	if !strings.Contains(md, "- Audit entries: 3 (refused 3)") {
		t.Errorf("markdown lacks the audit count: %.400s", md)
	}
	if got := field(out["candidates"], "count"); !reflect.DeepEqual(got, []any{3.0}) {
		t.Errorf("counts = %v", got)
	}
	for _, n := range []int{2, 4, 5} {
		if !strings.Contains(md, fmt.Sprintf("\n- unreadable audit line %d: UnicodeDecodeError", n)) {
			t.Errorf("markdown does not name unreadable line %d", n)
		}
	}
	if !strings.Contains(md, "- Incomplete: 3 audit line(s) are unreadable and left out of this draft") {
		t.Error("markdown lacks the incomplete line")
	}
	if strings.Contains(md, "unreadable audit line 1:") {
		t.Error("markdown names line 1 unreadable")
	}
}

func TestRetroDraftNoSecretReachesTheReply(t *testing.T) {
	c := newDraftCase(t, "test_no_secret_reaches_the_reply")
	c.audit(auditRow("2026-10-01T10:00:00+00:00", "n4_connect", "error", "BoxError: login failed for "+retroCanary,
		nil, obj("password", "***")),
		auditRow("2026-10-01T10:00:01+00:00", "n4_set_slot", "refused",
			"ord 'station:|slot:/"+retroCanary+"' is outside the write scope", nil, obj("credential", "***")))
	out := c.sameDraft(Args{"since": retroWhole})
	if strings.Contains(jsonText(out), retroCanary) {
		t.Error("the canary reached the reply")
	}
}

func containsAny(list []any, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
