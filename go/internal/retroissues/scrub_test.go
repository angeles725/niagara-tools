package retroissues

import (
	"strings"
	"testing"
)

// A dedup search query is rebuilt from sorted lowercased tokens (here "... crash northwind"), so it can match a rule
// the body ("crash for northwind") does not: it is scrubbed too, and a hit refuses the delta before any search
// leaves the machine.
func TestSearchQueryIsScrubbed(t *testing.T) {
	e := newEnv(t)
	e.write(e.denylist, "crash\\s+northwind\n")
	p := e.kitRetro("2026-02-01-alpha.md", "## Deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n"+
		"| Δ1 | mcp-n4 crash for northwind, traders | toolbelt/x.sh | t |\n")
	code, out, errs := e.run(p)
	for _, c := range e.gh.calls {
		if strings.Contains(strings.ToLower(c), "northwind") {
			t.Errorf("a gh call carries the denied words: %q", c)
		}
	}
	if code != 1 || !strings.Contains(out, "refused: retro alpha Δ1: search query matches denylist rule #1") {
		t.Errorf("exit %d, out %q, stderr %q", code, out, errs)
	}
}

func TestAnAliasThatIsADeniedWordIsRefusedUnprinted(t *testing.T) {
	e := newEnv(t)
	p := e.outRetro("2026-02-01-sample.md", alphaRetro)
	code, out, errs := e.run("--alias", "sample=SecretWord", p)
	if code != 3 || strings.Contains(strings.ToLower(out+errs), "secretword") || !strings.Contains(errs, "--alias value matches denylist rule #2") {
		t.Errorf("exit %d, out %q, stderr %q", code, out, errs)
	}
	if len(e.gh.calls) != 0 {
		t.Errorf("gh was called: %q", e.gh.calls)
	}
}

func TestDupThresholdRejectsNaN(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	for _, v := range []string{"NaN", "nan", "+Inf"} {
		if code, _, errs := e.run("--dup-threshold", v, p); code == 0 || !strings.Contains(errs, "--dup-threshold") {
			t.Errorf("%s: exit %d, stderr %q", v, code, errs)
		}
	}
}

// Two retros under one alias would share every marker: the second's deltas would read as already filed.
func TestTwoStemsCannotShareAnAlias(t *testing.T) {
	e := newEnv(t)
	p := e.outRetro("2026-02-01-sample.md", alphaRetro)
	q := e.outRetro("2026-02-02-other.md", alphaRetro)
	code, _, errs := e.run("--alias", "sample=ALPHA", "--alias", "other=ALPHA", p, q)
	if code != 3 || !strings.Contains(errs, "--alias ALPHA names two retros") || len(e.gh.calls) != 0 {
		t.Errorf("exit %d, stderr %q, calls %q", code, errs, e.gh.calls)
	}
}

// A run that only comments (every delta a duplicate) still makes sure the labels exist before the first write:
// the comment adds the retro-delta label, which a fresh repository lacks.
func TestDuplicateOnlyRunEnsuresTheLabelsFirst(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", slotsRetro)
	e.gh.searched = issueJSON(30, "OPEN", "mcp-n4: n4_read_slots drops a key", `The n4_read_slots filter loses the key when the slot name is long (go/internal/mcpn4/slots.go)`)
	if code, out, _ := e.run("--apply", p); code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	label, comment := -1, -1
	for i, c := range e.gh.calls {
		if strings.HasPrefix(c, "label list") && label < 0 {
			label = i
		}
		if strings.HasPrefix(c, "issue comment") && comment < 0 {
			comment = i
		}
	}
	if label < 0 || comment < 0 || label > comment {
		t.Errorf("labels must be ensured before the first comment; calls %q", e.gh.calls)
	}
}

// The closing line of --apply counts what gh created, not what was planned.
func TestApplySummaryCountsOnlyWhatWasCreated(t *testing.T) {
	e := newEnv(t)
	e.gh.createErr = errGH
	code, out, _ := e.run("--apply", e.kitRetro("2026-02-01-alpha.md", alphaRetro))
	if code != 1 || !strings.Contains(out, "retro-issues: 0 created") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

// With no denylist nothing can be scrubbed, so a dry run sends no search term to GitHub: the dedup falls back to the
// markers and the labeled issues it already listed, and says so (retro R17 delta D3).
func TestDryRunWithoutADenylistSendsNoSearchTerms(t *testing.T) {
	e := newEnv(t)
	e.denylist += ".missing"
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	code, out, errs := e.run(p)
	for _, c := range e.gh.calls {
		if strings.Contains(c, "--search=") {
			t.Errorf("a gh call carries search terms without a denylist: %q", c)
		}
	}
	want := "note: dedup search skipped: no denylist on this machine, so no search term is sent to GitHub\n"
	if code != 0 || strings.Count(out, want) != 1 || !strings.Contains(out, "scrub: no denylist") {
		t.Errorf("exit %d, out %q, stderr %q", code, out, errs)
	}
}

func TestDryRunWithADenylistStillSearches(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	if code, out, _ := e.run(p); code != 0 || strings.Contains(out, "dedup search skipped") || !searched(e.gh) {
		t.Errorf("exit %d, out %q, calls %q", code, out, e.gh.calls)
	}
}

func searched(g *fakeGH) bool {
	for _, c := range g.calls {
		if strings.Contains(c, "--search=") {
			return true
		}
	}
	return false
}
