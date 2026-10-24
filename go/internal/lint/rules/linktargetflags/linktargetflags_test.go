package linktargetflags

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-link-target-flags.bats run on a readable, non-empty tree and every edge tree,
// recorded from the bash lint (origin/main 8a1c26f) on copies of its root and map.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 54) }

// TestEmptyDir: an empty root directory (the golden format cannot hold one) is the K20 ERROR row, exit 3.
func TestEmptyDir(t *testing.T) {
	dir := t.TempDir()
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if want := "ERROR  " + ID + "  " + dir + "  no Java sources found (K20: wrong path or empty scaffold)\n"; code != 3 || o.String() != want || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, o.String(), e.String(), want)
	}
}

// TestLocked is LTF-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_or_exit did (text recorded from the bash lint), even beside a file that FAILs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class A {\n// link-in\n@NiagaraProperty(name = \"a\", flags = Flags.READONLY)\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	want := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n"
	if code != 3 || o.String() != "" || e.String() != want {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, o.String(), e.String(), want)
	}
}
