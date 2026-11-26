package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fdRun runs report-module --lint-only on a one-artifact module; args follow the root.
func fdRun(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	return fdRunAt(t, fdRoot(t), append([]string{"--lint-only"}, args...)...)
}

// fdRoot is a one-artifact module root.
func fdRoot(t *testing.T) string {
	t.Helper()
	return moduleTree(t, map[string]string{"A-rt/src/a/A.java": "class A {}\n"})
}

// fdRunAt runs report-module on the module root with every member a no-op stub (the golden cases'
// toolbelt/.stub-rest) and returns the exit code, stdout and stderr. args follow the root.
func fdRunAt(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	if testing.Short() {
		t.Skip("runs toolbelt members as subprocesses")
	}
	repo, err := filepath.Abs("../../..")
	must(t, err)
	dir := t.TempDir()
	tb := filepath.Join(dir, "toolbelt")
	must(t, os.MkdirAll(tb, 0o755))
	must(t, os.WriteFile(filepath.Join(tb, ".stub-rest"), nil, 0o644))
	toolbeltCopy(t, repo, dir)
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		return os.Getenv(k)
	}}
	code := mainEnv(append([]string{root}, args...), s, os.Environ())
	return code, o.String(), e.String()
}

// fdFile writes a feature document under a temp dir and returns its path.
func fdFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "feature.md")
	must(t, os.WriteFile(p, []byte(body), 0o644))
	return p
}

const (
	fdTier     = "# Feature: x (x)\n\nLocator: `odd/tasks/x.md` · base `main` · tier `P2` and the evidence.\n\n"
	fdDecided  = "## Behavior decisions\n\n| # | Question | Answer |\n|---|---|---|\n| B1 | Save on stop? | yes |\n\n"
	fdSkipped  = "## Checks skipped\nnone\n"
	fdUndecide = "## Behavior decisions\n\n| # | Question | Answer |\n|---|---|---|\n| B1 | | |\n\n"
)

// fdRows are the feature-doc rows of an output, in order.
func fdRows(out string) []string {
	var rows []string
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "  ADVISORY  feature-doc  ") {
			rows = append(rows, ln)
		}
	}
	return rows
}

// fdWant checks that out holds exactly one feature-doc row per entry of want, each containing its text, and the
// summary counts them as ADVISORY.
func fdWant(t *testing.T, out string, want ...string) {
	t.Helper()
	rows := fdRows(out)
	if len(rows) != len(want) {
		t.Fatalf("%d feature-doc rows, want %d:\n%s", len(rows), len(want), out)
	}
	for i, w := range want {
		if !strings.HasPrefix(rows[i], "(module)  ADVISORY  feature-doc  ") || !strings.Contains(rows[i], w) {
			t.Errorf("row %d = %q, want it to contain %q", i, rows[i], w)
		}
	}
	if sum := " · " + itoa(len(want)) + " ADVISORY  ->  "; !strings.Contains(out, sum) {
		t.Errorf("summary lacks %q:\n%s", sum, out)
	}
}

// S1: no tier line -> one ADVISORY row naming the tier.
func TestFeatureDocMissingTierRow(t *testing.T) {
	code, out, _ := fdRun(t, "--feature-doc", fdFile(t, "# Feature: x\n\n"+fdDecided+fdSkipped))
	if code != ExitClean {
		t.Errorf("exit %d, want %d:\n%s", code, ExitClean, out)
	}
	fdWant(t, out, "no tier line")
}

// S2: no `## Behavior decisions` heading -> one row naming the heading.
func TestFeatureDocMissingDecisionsHeadingRow(t *testing.T) {
	_, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+fdSkipped))
	fdWant(t, out, "no `## Behavior decisions` heading")
}

// S3: the heading with no answered row and no "none" (the template's empty B1 row) -> one row about the content.
func TestFeatureDocEmptyDecisionsRow(t *testing.T) {
	_, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+fdUndecide+fdSkipped))
	fdWant(t, out, "`## Behavior decisions` has no row and no \"none\"")
}

// S3b: "none" is content, with or without a reason after it.
func TestFeatureDocNoneDecisionsIsContent(t *testing.T) {
	for _, body := range []string{"None\n", "none - cosmetic copy change\n"} {
		_, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+"## Behavior decisions\n"+body+"\n"+fdSkipped))
		fdWant(t, out)
	}
}

// S4: no `## Checks skipped` heading -> one row naming it.
func TestFeatureDocMissingChecksSkippedRow(t *testing.T) {
	_, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+fdDecided))
	fdWant(t, out, "no `## Checks skipped` heading")
}

// S4b: the heading is enough even when it says none (and a heading with nothing under it too).
func TestFeatureDocChecksSkippedNoneIsPresent(t *testing.T) {
	for _, tail := range []string{"## Checks skipped\nnone\n", "## Checks skipped\n"} {
		_, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+fdDecided+tail))
		fdWant(t, out)
	}
}

// S5: all three present -> no feature-doc row and a 0 ADVISORY summary.
func TestFeatureDocAllPresentNoRows(t *testing.T) {
	code, out, _ := fdRun(t, "--feature-doc", fdFile(t, fdTier+fdDecided+fdSkipped))
	if code != ExitClean || len(fdRows(out)) != 0 || !strings.Contains(out, " · 0 ADVISORY  ->  CLEAN") {
		t.Errorf("exit %d, want %d with no feature-doc row:\n%s", code, ExitClean, out)
	}
}

// S6: every item missing -> one row each, in the order tier, decisions, checks skipped.
func TestFeatureDocAllMissingRowsInOrder(t *testing.T) {
	_, out, _ := fdRun(t, "--feature-doc", fdFile(t, "# Feature: x\n\n## Specs\n- S1. a\n"))
	fdWant(t, out, "no tier line", "no `## Behavior decisions` heading", "no `## Checks skipped` heading")
}
