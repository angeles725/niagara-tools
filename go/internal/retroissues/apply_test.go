package retroissues

import (
	"strings"
	"testing"
)

func TestScrubRefusesAnIssueNamingTheFieldNotTheWord(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", "## Deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n"+
		"| Δ1 | n4_foo fails at the secretword site | toolbelt/x.sh | t |\n"+
		"| Δ2 | n4_bar ok | toolbelt/secretword.sh | t |\n"+
		"| Δ3 | n4_baz is fine | toolbelt/y.sh | t |\n")
	code, out, errs := e.run("--apply", p)
	if code != 1 {
		t.Fatalf("exit %d, want 1; stderr %q", code, errs)
	}
	all := out + errs
	if strings.Contains(all, "secretword") {
		t.Errorf("the refused word was printed:\n%s", all)
	}
	for _, want := range []string{"refused: retro alpha Δ1: title matches denylist rule #2", "refused: retro alpha Δ2: body matches denylist rule #2"} {
		if !strings.Contains(all, want) {
			t.Errorf("missing %q in:\n%s", want, all)
		}
	}
	if e.gh.called("issue create") != 1 || !strings.Contains(strings.Join(e.gh.calls, "\n"), "n4_baz") {
		t.Errorf("only the clean issue may be created; calls %q", e.gh.calls)
	}
	// nothing leaves the machine before the scrub: no search carries the refused text either
	for _, c := range e.gh.calls {
		if strings.Contains(c, "secretword") {
			t.Errorf("a gh call carries the refused word: %q", c)
		}
	}
}

func TestDryRunAlsoScrubs(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", "## Deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n| Δ1 | n4_foo fails at SecretWord | x | t |\n")
	if code, out, _ := e.run(p); code != 1 || !strings.Contains(out, "refused: retro alpha Δ1") {
		t.Errorf("dry run: exit %d, out %q", code, out)
	}
}

func TestApplyNeedsADenylist(t *testing.T) {
	e := newEnv(t)
	e.denylist = e.denylist + ".missing"
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	code, _, errs := e.run("--apply", p)
	if code != 3 || !strings.Contains(errs, "denylist") || e.gh.called("issue create") != 0 {
		t.Errorf("apply without a denylist: exit %d, stderr %q, calls %q", code, errs, e.gh.calls)
	}
	if code, out, _ := e.run(p); code != 0 || !strings.Contains(out, "scrub: no denylist") {
		t.Errorf("a dry run without a denylist goes on and says so: exit %d, out %q", code, out)
	}
}

func TestApplyCreatesLabelsIssuesAndWritesBack(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	code, out, errs := e.run("--apply", "--write-back", p)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	for _, want := range []string{"created #101: mcp-n4: n4_read_slots drops a key", "created #102: build-n4-module: A lint for the module descriptor"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if e.gh.called("label create area:mcp-n4") != 1 || e.gh.called("label create area:build-n4-module") != 1 || e.gh.called("label create retro-delta") != 0 {
		t.Errorf("labels: %q", e.gh.calls)
	}
	var create string
	for i, c := range e.gh.calls {
		if strings.HasPrefix(c, "issue create") && strings.Contains(c, "n4_read_slots") {
			create = c + "\n" + e.gh.stdins[i]
		}
	}
	for _, want := range []string{"--label retro-delta", "--label area:mcp-n4", "<!-- retro-delta: alpha Δ2 -->"} {
		if !strings.Contains(create, want) {
			t.Errorf("create call lacks %q:\n%s", want, create)
		}
	}
	text := e.read(p)
	if !strings.Contains(text, "is long #101 |") || !strings.Contains(text, "module descriptor #102 |") || strings.Contains(text, "already shipped #") {
		t.Errorf("write-back:\n%s", text)
	}
}

func TestOutsideRetroPrintsTheMappingAndIsNeverRewritten(t *testing.T) {
	e := newEnv(t)
	body := "## Deltas\n| # | Proposed change | Target | Evidence |\n|---|---|---|---|\n| B1 | n4_foo is broken | tool | seen |\n"
	p := e.outRetro("2026-02-02-outsidestem.md", body)
	code, out, errs := e.run("--apply", "--write-back", "--alias", "outsidestem=ALPHA", p)
	if code != 0 || !strings.Contains(out, "mapping: ALPHA ΔB1 -> #101") || e.read(p) != body || strings.Contains(out+errs, "outsidestem") {
		t.Errorf("exit %d\nout %q\nerr %q\nfile changed: %v", code, out, errs, e.read(p) != body)
	}
}

func TestApplyGHFailures(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	e.gh.listErr = errGH
	if code, _, errs := e.run("--apply", p); code != 1 || !strings.Contains(errs, "not logged in") || e.gh.called("issue create") != 0 {
		t.Errorf("an unreadable issue list must stop an apply: exit %d, stderr %q, calls %q", code, errs, e.gh.calls)
	}
	if code, out, _ := e.run(p); code != 0 || !strings.Contains(out, "dedup unchecked") {
		t.Errorf("a dry run says the dedup is unchecked: exit %d, out %q", code, out)
	}
	e2 := newEnv(t)
	e2.gh.createErr = errGH
	if code, _, errs := e2.run("--apply", e2.kitRetro("2026-02-01-alpha.md", alphaRetro)); code != 1 || !strings.Contains(errs, "not logged in") {
		t.Errorf("a failed create: exit %d, stderr %q", code, errs)
	}
}

func TestCheckListsOpenRetroDeltaIssues(t *testing.T) {
	e := newEnv(t)
	e.gh.labeled = `[{"number":5,"title":"mcp-n4: a","state":"OPEN"},{"number":6,"title":"mcp-n4: b","state":"CLOSED"}]`
	code, out, _ := e.run("--check")
	if code != 1 || !strings.Contains(out, "open #5: mcp-n4: a") || strings.Contains(out, "#6") || !strings.Contains(out, "1 open") {
		t.Errorf("exit %d, out %q", code, out)
	}
	e.gh.labeled = `[{"number":6,"title":"x","state":"CLOSED"}]`
	if code, out, _ := e.run("--check"); code != 0 || !strings.Contains(out, "0 open") {
		t.Errorf("none open: exit %d, out %q", code, out)
	}
	e.gh.listErr = errGH
	if code, _, errs := e.run("--check"); code != 3 || !strings.Contains(errs, "not logged in") {
		t.Errorf("gh down must not read as clean: exit %d, stderr %q", code, errs)
	}
}
