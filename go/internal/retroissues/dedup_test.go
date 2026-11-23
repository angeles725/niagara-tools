package retroissues

import (
	"strconv"
	"strings"
	"testing"
)

// One retro, one delta: the unit every dedup test varies the remote around.
const slotsRetro = "## Deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n" +
	"| Δ1 | **n4_read_slots drops a key.** The filter loses the key when the slot name is long | `go/internal/mcpn4/slots.go` | t |\n"

func issueJSON(n int, state, title, body string) string {
	return `[{"number":` + strconv.Itoa(n) + `,"title":"` + title + `","state":"` + state + `","body":"` + body + `","labels":[],"comments":[]}]`
}

func TestExactMarkerSkips(t *testing.T) {
	for _, state := range []string{"OPEN", "CLOSED"} {
		e := newEnv(t)
		p := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
		e.gh.labeled = issueJSON(12, state, "mcp-n4: x", `text\n<!-- retro-delta: alpha Δ1 -->`)
		code, out, _ := e.run("--apply", p)
		want := "exists #12: retro alpha Δ1"
		if state == "CLOSED" {
			want = "exists #12 (closed): retro alpha Δ1"
		}
		if code != 0 || !strings.Contains(out, want) || e.gh.called("issue create") != 0 || e.gh.called("issue comment") != 0 {
			t.Errorf("%s: exit %d, out %q, calls %q", state, code, out, e.gh.calls)
		}
	}
}

func TestExactMarkerInACommentSkips(t *testing.T) {
	e := newEnv(t)
	e.gh.labeled = `[{"number":12,"title":"t","state":"OPEN","body":"b","labels":[],"comments":[{"body":"also\n<!-- retro-delta: alpha Δ1 -->"}]}]`
	_, out, _ := e.run("--apply", e.kitRetro("2026-02-01-alpha.md", slotsRetro))
	if !strings.Contains(out, "exists #12") || e.gh.called("issue comment") != 0 {
		t.Errorf("a rerun after a duplicate comment must skip:\n%s", out)
	}
}

func TestProbableDuplicateOpenIsCommentedNotCreated(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	e.gh.searched = issueJSON(30, "OPEN", "mcp-n4: n4_read_slots drops a key", `The n4_read_slots filter loses the key when the slot name is long (go/internal/mcpn4/slots.go)`)
	code, out, _ := e.run(p)
	if code != 0 || !strings.Contains(out, `probable duplicate of #30 (open) "mcp-n4: n4_read_slots drops a key"`) || !strings.Contains(out, "retro alpha Δ1") || strings.Contains(out, "create:") {
		t.Errorf("dry run: exit %d\n%s", code, out)
	}
	if !strings.Contains(strings.Join(e.gh.calls, "\n"), "--search=") || !strings.Contains(strings.Join(e.gh.calls, "\n"), "n4_read_slots") {
		t.Errorf("no search by key terms: %q", e.gh.calls)
	}
	code, out, _ = e.run("--apply", "--write-back", p)
	if code != 0 || e.gh.called("issue create") != 0 || e.gh.called("issue comment 30") != 1 || e.gh.called("issue edit 30 --add-label retro-delta") != 1 {
		t.Fatalf("apply: exit %d, out %q, calls %q", code, out, e.gh.calls)
	}
	var comment string
	for i, c := range e.gh.calls {
		if strings.HasPrefix(c, "issue comment 30") {
			comment = e.gh.stdins[i]
		}
	}
	if !strings.Contains(comment, "also proposed by retro alpha Δ1") || !strings.Contains(comment, "<!-- retro-delta: alpha Δ1 -->") {
		t.Errorf("comment:\n%s", comment)
	}
	if !strings.Contains(e.read(p), "slot name is long #30 |") {
		t.Errorf("write-back did not point the row at #30:\n%s", e.read(p))
	}
}

func TestDuplicateAlreadyClosedIsReportedAndNeverReopened(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	e.gh.searched = issueJSON(31, "CLOSED", "mcp-n4: n4_read_slots drops a key", `The n4_read_slots filter loses the key when the slot name is long`)
	code, out, _ := e.run("--apply", p)
	if code != 0 || !strings.Contains(out, `already fixed in #31 (closed) "mcp-n4: n4_read_slots drops a key": retro alpha Δ1`) {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if e.gh.called("issue create") != 0 || e.gh.called("issue reopen") != 0 || strings.Contains(strings.Join(e.gh.calls, "\n"), "--state open") {
		t.Errorf("a closed duplicate must not be reopened or recreated: %q", e.gh.calls)
	}
	if e.gh.called("issue comment 31") != 1 {
		t.Errorf("the marker comment that lets the fold audit count it is missing: %q", e.gh.calls)
	}
}

func TestNewOverridesTheSemanticMatchButNotTheMarker(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	e.gh.searched = issueJSON(30, "OPEN", "mcp-n4: n4_read_slots drops a key", `The n4_read_slots filter loses the key when the slot name is long`)
	code, out, _ := e.run("--apply", "--new", "1", p)
	if code != 0 || e.gh.called("issue create") != 1 || strings.Contains(out, "probable duplicate of") {
		t.Errorf("--new 1: exit %d, out %q, calls %q", code, out, e.gh.calls)
	}
	e2 := newEnv(t)
	e2.gh.labeled = issueJSON(12, "OPEN", "x", `<!-- retro-delta: alpha Δ1 -->`)
	_, out, _ = e2.run("--apply", "--new", "alpha:1", e2.kitRetro("2026-02-01-alpha.md", slotsRetro))
	if !strings.Contains(out, "exists #12") || e2.gh.called("issue create") != 0 {
		t.Errorf("--new must not bypass the exact marker:\n%s", out)
	}
}

func TestDifferentChangeIsNotADuplicate(t *testing.T) {
	e := newEnv(t)
	e.gh.searched = issueJSON(40, "OPEN", "mcp-n4: history windows", `n4_read_history needs a window argument for the histories`)
	_, out, _ := e.run(e.kitRetro("2026-02-01-alpha.md", slotsRetro))
	if !strings.Contains(out, "create: mcp-n4: n4_read_slots drops a key") || strings.Contains(out, "probable duplicate of") {
		t.Errorf("unrelated issue matched:\n%s", out)
	}
}

func TestTwoRetrosDescribingOneChangeBecomeOneIssue(t *testing.T) {
	e := newEnv(t)
	a := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	b := e.outRetro("2026-02-02-outsidestem.md", "## Deltas\n| # | Proposed change | Target | Evidence |\n|---|---|---|---|\n"+
		"| B7 | n4_read_slots filter loses the key when the slot name is long | go/internal/mcpn4/slots.go | seen on a station |\n"+
		"| B8 | a different thing about --max-rows in n4_list_batches | tool | seen |\n")
	code, out, errs := e.run("--apply", "--write-back", "--alias", "outsidestem=ALPHA", a, b)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if e.gh.called("issue create") != 2 {
		t.Fatalf("want 2 issues (one grouped, one separate), calls %q", e.gh.calls)
	}
	var grouped string
	for i, c := range e.gh.calls {
		if strings.HasPrefix(c, "issue create") && strings.Contains(e.gh.stdins[i], "retro alpha Δ1") {
			grouped = e.gh.stdins[i]
		}
	}
	for _, want := range []string{"**Source:** retro alpha Δ1", "retro ALPHA ΔB7", "<!-- retro-delta: alpha Δ1 -->", "<!-- retro-delta: ALPHA ΔB7 -->", "seen on a station"} {
		if !strings.Contains(grouped, want) {
			t.Errorf("grouped issue lacks %q:\n%s", want, grouped)
		}
	}
	if !strings.Contains(out, "mapping: ALPHA ΔB7 -> #101") || !strings.Contains(out, "mapping: ALPHA ΔB8 -> #102") || !strings.Contains(e.read(a), "slot name is long #101 |") {
		t.Errorf("mapping/write-back:\n%s\n%s", out, e.read(a))
	}
}

func TestNewKeepsTwoRetrosApart(t *testing.T) {
	e := newEnv(t)
	a := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	b := e.outRetro("2026-02-02-outsidestem.md", "## Deltas\n| # | Proposed change | Target | Evidence |\n|---|---|---|---|\n| B7 | n4_read_slots filter loses the key when the slot name is long | go/internal/mcpn4/slots.go | seen |\n")
	if code, _, errs := e.run("--apply", "--new", "ALPHA:B7", "--alias", "outsidestem=ALPHA", a, b); code != 0 || e.gh.called("issue create") != 2 {
		t.Errorf("exit %d (%s), calls %q", code, errs, e.gh.calls)
	}
}
