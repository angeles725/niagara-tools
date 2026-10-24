package arbitraryord

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-arbitrary-ord.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 26) }

// TestUnreadable is AO-finderr and AO-unreadable: a sub-directory the walk cannot enter and a file it cannot read
// are one stderr line each (awk_lint_list's and awk_lint_each's texts), the readable files still scanned, exit 3.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class A { void f(String q){ BOrd.make(q); } }\n"
	a, u, locked := filepath.Join(dir, "a", "A.java"), filepath.Join(dir, "a", "U.java"), filepath.Join(dir, "locked")
	for _, p := range []string{a, u, filepath.Join(locked, "B.java")} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{u, locked} {
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	row := "WARN  " + ID + "  " + a + ":1  BOrd.make from a variable: " + src
	errs := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		ID + ": cannot scan " + u + ": Permission denied\n"
	if code != 3 || o.String() != row || e.String() != errs {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), row, errs)
	}
}
