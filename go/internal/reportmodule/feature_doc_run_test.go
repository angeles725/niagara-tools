package reportmodule

import (
	"path/filepath"
	"strings"
	"testing"
)

// S7: an unreadable file -> one ADVISORY row saying so, nothing else about the document.
func TestFeatureDocUnreadableOneRow(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope.md")
	code, out, _ := fdRun(t, "--feature-doc", missing)
	if code != ExitClean {
		t.Errorf("exit %d, want %d:\n%s", code, ExitClean, out)
	}
	fdWant(t, out, "feature doc not readable: "+missing)
}

// S8: without --feature-doc nothing new is printed.
func TestFeatureDocAbsentNoRows(t *testing.T) {
	_, out, _ := fdRun(t)
	if len(fdRows(out)) != 0 || strings.Contains(out, "feature-doc") || !strings.Contains(out, " · 0 ADVISORY  ->  CLEAN") {
		t.Errorf("a run without --feature-doc printed a feature-doc row or changed the summary:\n%s", out)
	}
}

// S9: the rows are advisory: the exit code and the verdict equal those of the run without the option, and so does
// every other line of stdout and stderr (the summary differs only in its ADVISORY count).
func TestFeatureDocDoesNotChangeExitOrVerdict(t *testing.T) {
	root := fdRoot(t)
	code0, out0, err0 := fdRunAt(t, root, "--lint-only")
	code1, out1, err1 := fdRunAt(t, root, "--lint-only", "--feature-doc", fdFile(t, "# Feature: x\n"))
	if code1 != code0 || err1 != err0 {
		t.Errorf("exit %d / stderr %q with the option, %d / %q without", code1, err1, code0, err0)
	}
	if len(fdRows(out1)) != 3 {
		t.Fatalf("want 3 feature-doc rows:\n%s", out1)
	}
	strip := func(out string) string {
		var keep []string
		for _, ln := range strings.Split(out, "\n") {
			if !strings.Contains(ln, "  ADVISORY  feature-doc  ") {
				keep = append(keep, ln)
			}
		}
		return strings.NewReplacer(" · 3 ADVISORY  ->", " · 0 ADVISORY  ->").Replace(strings.Join(keep, "\n"))
	}
	if strip(out1) != out0 {
		t.Errorf("stdout differs beyond the feature-doc rows and their count:\n--- without\n%s--- with\n%s", out0, out1)
	}
}

// S10: --feature-doc is a lint option: --ready refuses it like the other lint-run options.
func TestFeatureDocIsNotAReadyOption(t *testing.T) {
	code, _, errOut := fdRunAt(t, fdRoot(t), "--ready", "--feature-doc", fdFile(t, fdTier))
	if code != ExitUsage || !strings.HasPrefix(errOut, "report-module: --feature-doc is not a --ready option\n") {
		t.Errorf("exit %d, stderr %q", code, errOut)
	}
}
