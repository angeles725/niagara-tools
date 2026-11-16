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

func reasons(t *testing.T, root string, baseline map[string]int) map[string]string {
	t.Helper()
	findings, err := Check(root, baseline)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range findings {
		got[f.Path] = f.Reason
	}
	return got
}

func TestAFileOverTheLimitThatIsNotListedIsAFinding(t *testing.T) {
	root := t.TempDir()
	writeLines(t, root, "a/small.go", Limit)
	writeLines(t, root, "a/big.go", Limit+1)
	got := reasons(t, root, nil)
	if len(got) != 1 || !strings.Contains(got["a/big.go"], "over the limit") {
		t.Fatalf("findings = %v, want only a/big.go over the limit", got)
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
	writeLines(t, root, "fixed.go", Limit)
	got := reasons(t, root, map[string]int{"fixed.go": 300, "gone.go": 300})
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
	for _, f := range findings {
		t.Errorf("%s %s", f.Path, f.Reason)
	}
	if len(findings) > 0 {
		t.Log("the rule and how to fix a finding: go/internal/filesize/filesize.go and go/README.md")
	}
}
