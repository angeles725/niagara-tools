package retro

// Go twins of the test_retro.TestDerive tests on the candidates and the window
// (go_server.go_twin("retro", ...)).

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const canary = "pw-Zk93-unique-canary"

var b1, b2, b3, b4 = fmt.Sprintf("%032x", 1), fmt.Sprintf("%032x", 2), fmt.Sprintf("%032x", 3), fmt.Sprintf("%032x", 4)

// stateDir is the Fixture: a state dir the test appends audit and journal lines to.
type stateDir struct {
	t   *testing.T
	dir string
}

func newStateDir(t *testing.T) *stateDir { return &stateDir{t, filepath.Join(t.TempDir(), "state")} }

func (s *stateDir) audit(ts, tool, outcome, reason string, batchID any) {
	s.t.Helper()
	if err := journal.NewAuditLog(s.dir).Append(obj("ts", ts, "tool", tool, "batch_id", batchID,
		"dry_run", outcome == "planned", "outcome", outcome, "reason", reason, "args_redacted", obj())); err != nil {
		s.t.Fatal(err)
	}
}

// batch is journal_batch: an intent and, unless withResult is false, its result.
func (s *stateDir) batch(id, ts string, verdict any, inDoubt, withResult bool, ops ...any) {
	s.t.Helper()
	if ops == nil {
		ops = []any{obj("nm", "s")}
	}
	j := journal.New(s.dir)
	rows := []*pyjson.Object{obj("batch_id", id, "ts", ts, "tool", "n4_set_slot", "ops", ops, "phase", "intent",
		"station_name", "LLM", "inverse_plan", []any{}, "targets", obj())}
	if withResult {
		res := obj("batch_id", id, "ts", ts, "phase", "result", "accepted", nil, "inverse", []any{}, "verdict", verdict)
		if inDoubt {
			res.Set("in_doubt", true)
		}
		rows = append(rows, res)
	}
	for _, r := range rows {
		if err := j.Append(r); err != nil {
			s.t.Fatal(err)
		}
	}
}

func (s *stateDir) draft(since string, obs ...*pyjson.Object) *pyjson.Object {
	s.t.Helper()
	out, err := draft(s.dir, "LLM", "2026-10-01", since, obs, sinceFilter)
	if err != nil {
		s.t.Fatal(err)
	}
	return out
}

// keyed maps candidate keys to candidates; a key the test asks for that is missing fails it.
type keyed struct {
	t *testing.T
	m map[string]*pyjson.Object
}

func (k keyed) get(key string) *pyjson.Object {
	k.t.Helper()
	c := k.m[key]
	if c == nil {
		k.t.Fatalf("no candidate %s among %d", key, len(k.m))
	}
	return c
}

func candidates(t *testing.T, out *pyjson.Object) keyed {
	m := map[string]*pyjson.Object{}
	for _, c := range get(out, "candidates").([]any) {
		m[get(c.(*pyjson.Object), "key").(string)] = c.(*pyjson.Object)
	}
	return keyed{t, m}
}

func evidence(c *pyjson.Object) string {
	var parts []string
	for _, e := range get(c, "evidence").([]any) {
		parts = append(parts, e.(string))
	}
	return strings.Join(parts, "\n")
}

func count(c *pyjson.Object) int { return get(c, "count").(int) }

var everyReason = []struct{ reason, key string }{
	{"confirmation_token is missing or malformed: run a dry run first", "token_missing"},
	{"confirmation_token does not match this tool, these arguments", "token_mismatch"},
	{"confirmation_token expired: run the dry run again", "token_expired"},
	{"confirmation_token already used: run the dry run again", "token_reused"},
	{"station identity not verified: reconnect with n4_connect", "identity"},
	{"ord 'station:|slot:/X' is outside the write scope", "scope"},
	{"write budget exhausted: 2 writes already executed in this session", "budget"},
	{"refusing to write 'out' on its own: Status slots are written whole", "partial_status"},
	{"action 'foo' is not allowed in this version (allowed: set)", "unsupported"},
	{"type must look like module:Type, got 'x'", "unsupported"},
}

func TestEachRefusalReasonBecomesACandidateCitingAuditTimestamps(t *testing.T) {
	s := newStateDir(t)
	for i, r := range everyReason {
		s.audit(fmt.Sprintf("2026-10-01T10:00:%02d+00:00", i), "n4_set_slot", "refused", r.reason, nil)
	}
	cands := candidates(t, s.draft(""))
	for _, r := range everyReason {
		cands.get("refusal:" + r.key)
	}
	scope := cands.get("refusal:scope")
	if !strings.Contains(evidence(scope), "2026-10-01T10:00:05+00:00") || count(scope) != 1 {
		t.Errorf("scope %s", pyjson.Repr(scope))
	}
}

func TestSameReasonGroupsIntoOneCandidateWithACount(t *testing.T) {
	s := newStateDir(t)
	for i := 0; i < 3; i++ {
		s.audit(fmt.Sprintf("2026-10-01T10:00:0%d+00:00", i), "n4_create_link", "refused", "confirmation_token expired: run again", nil)
	}
	c := candidates(t, s.draft("")).get("refusal:token_expired")
	if count(c) != 3 || len(get(c, "evidence").([]any)) != 3 {
		t.Errorf("%s", pyjson.Repr(c))
	}
}

func TestReadbackVerdictsMismatchFailedUnverifiedCiteBatchIds(t *testing.T) {
	s := newStateDir(t)
	for i, v := range []struct{ id, verdict string }{{b1, "mismatch"}, {b2, "failed"}, {b3, "unverified"}, {b4, "verified"}} {
		s.batch(v.id, fmt.Sprintf("2026-10-01T11:0%d:00+00:00", i), v.verdict, false, true)
	}
	out := s.draft("")
	cands := candidates(t, out)
	for id, key := range map[string]string{b1: "verdict:mismatch", b2: "verdict:failed", b3: "verdict:unverified"} {
		if !strings.Contains(evidence(cands.get(key)), id) {
			t.Errorf("%s does not cite %s", key, id)
		}
	}
	if j, _ := pyjson.Marshal(get(out, "candidates"), pyjson.Options{}); strings.Contains(string(j), b4) {
		t.Errorf("a verified batch is a candidate: %s", j)
	}
}

func TestInDoubtBatchesAreFlaggedWhetherResultMissingOrFlagged(t *testing.T) {
	s := newStateDir(t)
	s.batch(b1, "2026-10-01T11:00:00+00:00", nil, false, false)
	s.batch(b2, "2026-10-01T11:01:00+00:00", "unverified", true, true)
	c := candidates(t, s.draft("")).get("in_doubt")
	if count(c) != 2 || !strings.Contains(evidence(c), b1) || !strings.Contains(evidence(c), b2) || get(c, "priority") != "HIGH" {
		t.Errorf("%s", pyjson.Repr(c))
	}
}

func TestBoxErrorsAreGroupedByOpNameFromTheJournal(t *testing.T) {
	s := newStateDir(t)
	s.batch(b1, "2026-10-01T11:00:00+00:00", nil, false, false, obj("nm", "a"), obj("nm", "s"))
	s.audit("2026-10-01T11:00:01+00:00", "n4_create_component", "error",
		"station call failed after the intent was journaled: batch "+b1+" is in-doubt", b1)
	s.audit("2026-10-01T11:05:00+00:00", "n4_set_slot", "error", "BoxError: HTTP 500", nil)
	cands := candidates(t, s.draft(""))
	cands.get("box_error:s")
	cands.get("box_error:n4_set_slot")
	if !strings.Contains(evidence(cands.get("box_error:a")), b1) {
		t.Errorf("%v", cands.m)
	}
}

func dangling(ts string, n int) *pyjson.Object {
	return obj("ts", ts, "ord", "station:|slot:/F", "count", n)
}

func TestDanglingObservationsBecomeACandidate(t *testing.T) {
	s := newStateDir(t)
	c := candidates(t, s.draft("", dangling("2026-10-01T12:00:00+00:00", 2), dangling("2026-10-01T12:05:00+00:00", 3),
		obj("ts", "2026-10-01T12:06:00+00:00", "tool", "n4_bql_query", "rows", 4))).get("dangling")
	items := get(c, "evidence").([]any)
	if count(c) != 2 || len(items) != 2 || !strings.Contains(evidence(c), "2026-10-01T12:00:00+00:00") ||
		!strings.Contains(items[1].(string), "found 3") {
		t.Errorf("%s", pyjson.Repr(c))
	}
}

func TestCountMeansTheNumberOfEvidenceItemsOnEveryCandidate(t *testing.T) {
	s := newStateDir(t)
	for i := 0; i < 2; i++ {
		s.audit(fmt.Sprintf("2026-10-01T10:00:0%d+00:00", i), "n4_set_slot", "refused", "ord 'x' is outside the write scope", nil)
	}
	s.batch(b1, "2026-10-01T11:00:00+00:00", "mismatch", false, true)
	cands := candidates(t, s.draft("", dangling("2026-10-01T12:00:00+00:00", 9)))
	if len(cands.m) != 3 {
		t.Fatalf("%v", cands.m)
	}
	for key, c := range cands.m {
		if count(c) != len(get(c, "evidence").([]any)) {
			t.Errorf("%s: count %d", key, count(c))
		}
	}
}

func TestTheThreeScopeReasonsAreOneCandidate(t *testing.T) {
	s := newStateDir(t)
	for i, r := range []string{"no --write-scope configured: every write is refused", "ord 'x' is not a plain station ORD",
		"ord 'x' is outside the write scope"} {
		s.audit(fmt.Sprintf("2026-10-01T10:00:0%d+00:00", i), "n4_set_slot", "refused", r, nil)
	}
	all := get(s.draft(""), "candidates").([]any)
	if len(all) != 1 || get(all[0].(*pyjson.Object), "key") != "refusal:scope" || count(all[0].(*pyjson.Object)) != 3 {
		t.Errorf("%s", pyjson.Repr(all))
	}
	rows := 0
	for _, r := range refusals {
		if r.key == "scope" {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("%d scope rows", rows)
	}
}

func TestDraftFiltersBySinceOnce(t *testing.T) {
	s := newStateDir(t)
	s.audit("2026-10-01T10:00:00+00:00", "n4_set_slot", "refused", "ord 'x' is outside the write scope", nil)
	calls := 0
	counting := func(since string) (keepFunc, error) { calls++; return sinceFilter(since) }
	if _, err := draft(s.dir, "LLM", "2026-10-01", "2026-10-01T08:00:00Z", nil, counting); err != nil || calls != 1 {
		t.Errorf("%d filters, %v", calls, err)
	}
}

func TestSinceFiltersOlderEntries(t *testing.T) {
	s := newStateDir(t)
	s.audit("2026-10-01T09:00:00+00:00", "n4_set_slot", "refused", "ord 'x' is outside the write scope", nil)
	s.batch(b1, "2026-10-01T09:00:00+00:00", "mismatch", false, true)
	if n := len(get(s.draft("2026-10-01T10:00:00Z"), "candidates").([]any)); n != 0 {
		t.Errorf("%d candidates after the window start", n)
	}
	if n := len(get(s.draft("2026-10-01T08:00:00+00:00"), "candidates").([]any)); n != 2 {
		t.Errorf("%d candidates before it", n)
	}
}

func TestBadSinceIsAValueError(t *testing.T) {
	s := newStateDir(t)
	_, err := draft(s.dir, "LLM", "2026-10-01", "yesterday", nil, sinceFilter)
	if ve, ok := err.(*pyjson.ValueError); !ok || ve.Msg != "not an ISO-8601 timestamp: 'yesterday'" {
		t.Errorf("%v", err)
	}
}

func TestNoSecretLeaksFromReasonsOrArgs(t *testing.T) {
	s := newStateDir(t)
	if err := journal.NewAuditLog(s.dir).Append(obj("ts", "2026-10-01T10:00:00+00:00", "tool", "n4_connect",
		"outcome", "error", "reason", "BoxError: login failed for "+canary, "args_redacted", obj("password", canary))); err != nil {
		t.Fatal(err)
	}
	s.audit("2026-10-01T10:00:01+00:00", "n4_set_slot", "refused", "ord 'station:|slot:/"+canary+"' is outside the write scope", nil)
	out := s.draft("")
	blob, _ := pyjson.Marshal(out, pyjson.Options{})
	if len(get(out, "candidates").([]any)) == 0 || strings.Contains(string(blob), canary) {
		t.Errorf("%s", blob)
	}
}

func TestMissingStateDirYieldsNoCandidates(t *testing.T) {
	out, err := draft(filepath.Join(t.TempDir(), "nope"), "LLM", "2026-10-01", "", nil, sinceFilter)
	if err != nil {
		t.Fatal(err)
	}
	if len(get(out, "candidates").([]any)) != 0 {
		t.Errorf("%s", pyjson.Repr(out))
	}
}
