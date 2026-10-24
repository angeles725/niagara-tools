package agentonshape

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-agent-on-shape.bats run on a tree the format can hold and every edge tree, recorded
// from the bash lint (origin/main ecfa381).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 26) }

// TestUnreadable is AOS-finderr and AOS-unreadable: a sub-directory the walk cannot enter and a file it cannot read
// are one stderr line each (awk_lint_list's and awk_lint_each's texts), the readable files still scanned, exit 3.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	for _, d := range []string{"a", "b", "locked"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, d, "module-include.xml"), []byte("<on type=\"Bad\"/>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	u, locked := filepath.Join(dir, "b", "module-include.xml"), filepath.Join(dir, "locked")
	for _, p := range []string{u, locked} {
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
		p := p
		t.Cleanup(func() { _ = os.Chmod(p, 0o755) })
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	row := "FAIL  " + ID + "  " + dir + "/a/module-include.xml:1  malformed agent-on target (need module:Type): Bad\n"
	errs := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		ID + ": cannot scan " + u + ": Permission denied\n"
	if code != 3 || o.String() != row || e.String() != errs {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), row, errs)
	}
}
