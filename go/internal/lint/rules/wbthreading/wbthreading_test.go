package wbthreading

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-wb-threading.bats run on a readable, non-empty tree and every edge tree, recorded
// from the bash lint (origin/main 1c2e425) on copies of its roots.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 34) }

// TestLocked is LW-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_nonempty_or_exit did (text recorded from the bash lint), even beside a file that WARNs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte("@AgentOn(types = \"baja:Component\")\nclass A {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, "--strict", dir}, cli.Stdio{Out: &o, Err: &e})
	want := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n"
	if code != 3 || o.String() != "" || e.String() != want {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, o.String(), e.String(), want)
	}
}
