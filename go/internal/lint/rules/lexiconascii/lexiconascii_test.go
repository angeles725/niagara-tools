package lexiconascii

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-lexicon-ascii.bats run on a tree the format can hold and every edge tree,
// recorded from the bash lint (origin/main e6581fd).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 26) }

// TestEmptyDir is LA5 on an empty directory (the golden format cannot hold one): no row, exit 0.
func TestEmptyDir(t *testing.T) {
	dir := t.TempDir()
	var o, e bytes.Buffer
	if code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e}); code != 0 || o.Len() != 0 || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 0 and nothing", code, o.String(), e.String())
	}
}

// TestLocked is LA-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_or_exit did (text recorded from the bash lint), even beside a file that FAILs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "Mod-rt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Mod-rt", "module.lexicon"), []byte("k=\xc3\xa9\n"), 0o644); err != nil {
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

// TestUnreadable: an unreadable lexicon is an env error, the other files still scanned (declared port
// difference: the bash read loop printed the shell's error and went on).
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.lexicon"), []byte("k=\xc3\xa9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	u := filepath.Join(dir, "u.lexicon")
	if err := os.WriteFile(u, []byte("k=x\n"), 0); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if code != 3 || o.String() != "FAIL  "+ID+"  a.lexicon:1  non-ASCII: 0xc3 0xa9\n" || !bytes.Contains(e.Bytes(), []byte(ID+": cannot scan "+u)) {
		t.Errorf("exit %d out %q err %q", code, o.String(), e.String())
	}
}
