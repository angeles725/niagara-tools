package uberjarapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-uberjar-api-conflict.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }

// TestUnreadable is UAC-finderr plus an unreadable file: the walk error is awk_lint_list's env line and the
// listed files are still scanned; an unreadable *.gradle.kts, which the bash grep 2>/dev/null skipped (a
// clean exit 0), is "cannot scan <path>: <reason>", exit 3.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "api(\"g:a:1\")\nuberjar(\"g:a:1\")\n"
	a, u, locked := filepath.Join(dir, "a", "a.gradle.kts"), filepath.Join(dir, "a", "u.gradle.kts"), filepath.Join(dir, "locked")
	for _, p := range []string{a, u, filepath.Join(locked, "b.gradle.kts")} {
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
	row := "WARN  " + ID + "  " + a + "  g:a is both uberjar()d and api()d\n"
	errs := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		ID + ": cannot scan " + u + ": Permission denied\n"
	if code != 3 || o.String() != row || e.String() != errs {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), row, errs)
	}
}
