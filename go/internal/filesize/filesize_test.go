package filesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLines(t *testing.T, root, rel string, n int) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Repeat("x\n", n)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// reasons returns the findings that fail the gate; warns returns the ones that are only reported.
func reasons(t *testing.T, root string, baseline map[string]int) map[string]string {
	t.Helper()
	return byPath(t, root, baseline, false)
}

func warns(t *testing.T, root string, baseline map[string]int) map[string]string {
	t.Helper()
	return byPath(t, root, baseline, true)
}

func byPath(t *testing.T, root string, baseline map[string]int, warn bool) map[string]string {
	t.Helper()
	findings, err := Check(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range findings {
		if f.Warn == warn {
			got[f.Path] = f.Reason
		}
	}
	return got
}

func TestAFileOverTheFailLimitThatIsNotListedFails(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/edge.go", 300)
	writeLines(t, root, "a/big.go", 301)
	got := reasons(t, root, nil)
	if len(got) != 1 || !strings.Contains(got["a/big.go"], "over the limit of 300: split it by responsibility") {
		t.Fatalf("failing findings = %v, want only a/big.go over the limit", got)
	}
}

func TestTheWarnBandStartsAbove200AndNeverFails(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/ok.go", 200)
	writeLines(t, root, "a/warn_low.go", 201)
	writeLines(t, root, "a/warn_high.go", 300)
	if got := reasons(t, root, nil); len(got) != 0 {
		t.Fatalf("failing findings = %v, want none for 200, 201 and 300 lines", got)
	}
	got := warns(t, root, nil)
	if len(got) != 2 || got["a/warn_low.go"] == "" || got["a/warn_high.go"] == "" {
		t.Fatalf("warnings = %v, want a/warn_low.go and a/warn_high.go only", got)
	}
	if !strings.Contains(got["a/warn_low.go"], "split by responsibility") {
		t.Fatalf("warning = %q, want the split-by-responsibility advice", got["a/warn_low.go"])
	}
}

func TestAFileOver300IsAFailureNotAWarning(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/big.go", 301)
	if got := warns(t, root, nil); len(got) != 0 {
		t.Fatalf("warnings = %v, want none: a file over 300 fails instead", got)
	}
}

func TestAListedFileGetsNoWarning(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "listed.go", 250)
	if got := warns(t, root, map[string]int{"listed.go": 300}); len(got) != 0 {
		t.Fatalf("warnings = %v, want none for a baseline-listed file", got)
	}
}

func TestAListedFileMayStayOrShrinkButNotGrow(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "same.go", 300)
	writeLines(t, root, "shrunk.go", 250)
	writeLines(t, root, "grew.go", 301)
	got := reasons(t, root, map[string]int{"same.go": 300, "shrunk.go": 300, "grew.go": 300})
	if len(got) != 1 || !strings.Contains(got["grew.go"], "grew from its recorded 300 lines to 301") {
		t.Fatalf("findings = %v, want only grew.go", got)
	}
}

func TestAStaleBaselineLineIsAFinding(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "fixed.go", 200)
	writeLines(t, root, "fixed201.go", 201)
	got := reasons(t, root, map[string]int{"fixed.go": 300, "fixed201.go": 300, "gone.go": 300})
	if _, stale := got["fixed201.go"]; stale {
		t.Fatalf("findings = %v, want a listed 201-line file left alone", got)
	}
	if !strings.Contains(got["fixed.go"], "within the limit") || !strings.Contains(got["gone.go"], "no longer exists") {
		t.Fatalf("findings = %v, want fixed.go and gone.go reported as stale", got)
	}
}

func TestTestFilesTestdataAndBinAreNotMeasured(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/big_test.go", 900)
	writeLines(t, root, "a/testdata/big.go", 900)
	writeLines(t, root, "bin/big.go", 900)
	writeLines(t, root, "a/notgo.txt", 900)
	if got := reasons(t, root, nil); len(got) != 0 {
		t.Fatalf("findings = %v, want none", got)
	}
}

func TestABaselineLineThatIsNotLinesThenPathIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "baseline.txt")
	if err := os.WriteFile(path, []byte("# comment\n\n300 a.go\nnot-a-number b.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadBaseline(path); err == nil || !strings.Contains(err.Error(), "baseline.txt:4") {
		t.Fatalf("err = %v, want a parse error naming line 4", err)
	}
}

// TestTheGoTreeKeepsItsFilesSmall is the gate: it measures this repository's go/ tree against baseline.txt.
func TestTheGoTreeKeepsItsFilesSmall(t *testing.T) {
	baseline, err := ReadBaseline("baseline.txt")
	if err != nil {
		t.Fatal(err)
	}
	findings, err := Check(filepath.Join("..", ".."), baseline)
	if err != nil {
		t.Fatal(err)
	}
	failed := 0
	for _, f := range findings {
		if f.Warn {
			t.Logf("WARN %s %s", f.Path, f.Reason)
			continue
		}
		failed++
		t.Errorf("%s %s", f.Path, f.Reason)
	}
	if failed > 0 {
		t.Log("the rule and how to fix a finding: go/internal/filesize/filesize.go and go/README.md")
	}
}
