package ktssrc

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSweepRealFiles tokenizes every *.gradle.kts under the KTSSRC_SWEEP_ROOTS roots (colon-separated; dot and
// *-worktrees directories skipped) and fails on any it cannot read: real client build scripts are well-formed
// Kotlin, so an error is a scanner bug. Skipped when the variable is unset.
func TestSweepRealFiles(t *testing.T) {
	roots := os.Getenv("KTSSRC_SWEEP_ROOTS")
	if roots == "" {
		t.Skip("KTSSRC_SWEEP_ROOTS not set")
	}
	n := 0
	for _, root := range strings.Split(roots, ":") {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && p != root && (strings.HasPrefix(d.Name(), ".") || strings.HasSuffix(d.Name(), "-worktrees")) {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(p, ".gradle.kts") {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if _, err := Tokens(string(data)); err != nil {
				t.Errorf("%s: %v", p, err)
			}
			n++
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%d files", n)
}
