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
