package retroissues

import (
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestDryRunPlansIssuesAndCallsNothingThatWrites(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	code, out, errs := e.run(p)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	for _, want := range []string{
		"create: mcp-n4: n4_read_slots drops a key",
		"create: build-n4-module: A lint for the module descriptor",
		"labels: retro-delta, area:mcp-n4",
		"labels: retro-delta, area:build-n4-module",
		"**Source:** retro alpha Δ2",
		"<!-- retro-delta: alpha Δ2 -->",
		"**Evidence:** `[ev: retro alpha Δ2]`",
		"The filter loses the key",
		"3 to create",
		"build-n4-module: Something with no area at all", // a retro inside the kit defaults to build-n4-module
		"1 skipped (applied)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dry run lacks %q:\n%s", want, out)
		}
	}
	for _, c := range e.gh.calls {
		if !strings.HasPrefix(c, "issue list") {
			t.Errorf("a dry run made a call that is not a read: %q", c)
		}
	}
	if strings.Contains(out, "already shipped") {
		t.Error("an APPLIED row became an issue")
	}
}

func TestTitleNeverExceedsOneHundredCharacters(t *testing.T) {
	e := newEnv(t)
	long := strings.Repeat("palabra ", 30)
	p := e.kitRetro("2026-02-01-alpha.md", "## Deltas\n| Δ | Delta | Target | Token |\n|---|---|---|---|\n| Δ1 | n4_foo "+long+" | x | y |\n")
	_, out, _ := e.run(p)
	for _, l := range strings.Split(out, "\n") {
		if t2, ok := strings.CutPrefix(l, "create: "); ok && utf8.RuneCountInString(t2) > 100 {
			t.Errorf("title is %d characters: %q", utf8.RuneCountInString(t2), t2)
		}
	}
	if !strings.Contains(out, "create: mcp-n4: n4_foo") {
		t.Errorf("no title planned:\n%s", out)
	}
}

func TestAreaFlagOverridesDetection(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	_, out, _ := e.run("--area", "build-n4-module", p)
	if strings.Contains(out, "create: mcp-n4:") || !strings.Contains(out, "create: build-n4-module: n4_read_slots drops a key") {
		t.Errorf("--area did not override:\n%s", out)
	}
	if code, _, errs := e.run("--area", "other", p); code != 3 || !strings.Contains(errs, "--area") {
		t.Errorf("a bad --area: exit %d, stderr %q", code, errs)
	}
}

func TestOnlyFiltersRows(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", alphaRetro)
	_, out, _ := e.run("--only", "2", p)
	if !strings.Contains(out, "n4_read_slots") || strings.Contains(out, "module descriptor") {
		t.Errorf("--only 2 kept the wrong rows:\n%s", out)
	}
}

func TestOutsideRetroNeedsAnAliasAndNeverShowsItsStem(t *testing.T) {
	e := newEnv(t)
	p := e.outRetro("2026-02-02-outsidestem.md", "## Deltas\n| # | Proposed change | Target | Evidence |\n|---|---|---|---|\n| B1 | n4_foo is broken | tool | seen |\n| B2 | something generic | nowhere | seen |\n")
	code, out, errs := e.run(p)
	if code != 3 || !strings.Contains(errs, "--alias") || strings.Contains(out+errs, "outsidestem") {
		t.Errorf("no alias: exit %d, stdout %q, stderr %q", code, out, errs)
	}
	code, out, errs = e.run("--alias", "outsidestem=ALPHA", p)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errs)
	}
	if !strings.Contains(out, "1 skipped (no area)") || strings.Contains(out, "something generic") {
		t.Errorf("a row with no area in a retro outside the kit must be skipped:\n%s", out)
	}
	if !strings.Contains(out, "**Source:** retro ALPHA ΔB1") || !strings.Contains(out, "<!-- retro-delta: ALPHA ΔB1 -->") || strings.Contains(out+errs, "outsidestem") {
		t.Errorf("the alias is not the only name shown:\n%s", out)
	}
	if code, _, errs := e.run("--alias", "outsidestem=bad alias", p); code != 3 || !strings.Contains(errs, "--alias") {
		t.Errorf("a bad alias: exit %d, stderr %q", code, errs)
	}
}

func TestRowThatAlreadyCarriesAnIssueIsTracked(t *testing.T) {
	e := newEnv(t)
	p := e.kitRetro("2026-02-01-alpha.md", strings.Replace(alphaRetro, "A lint for the module descriptor |", "A lint for the module descriptor #77 |", 1))
	_, out, _ := e.run(p)
	if !strings.Contains(out, "tracked #77: retro alpha Δ3") || strings.Contains(out, "create: build-n4-module: A lint") {
		t.Errorf("a row with #N was planned again:\n%s", out)
	}
}

func TestUsageErrors(t *testing.T) {
	e := newEnv(t)
	for name, args := range map[string][]string{
		"no retro":                 {},
		"unknown flag":             {"--nope", "x.md"},
		"write-back without apply": {"--write-back", e.kitRetro("2026-02-01-alpha.md", alphaRetro)},
		"missing file":             {filepath.Join(e.kit, "retros", "none.md")},
		"bad threshold":            {"--dup-threshold", "2", e.kitRetro("2026-02-01-alpha.md", alphaRetro)},
	} {
		if code, _, errs := e.run(args...); code != 3 || !strings.HasPrefix(errs, "retro-issues: ") {
			t.Errorf("%s: exit %d, stderr %q, want 3 and a retro-issues: message", name, code, errs)
		}
	}
}
