package statusparity

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-status-parity.bats run (LSP-locked aside: its locked directory is not a
// golden tree; TestLocked pins it) and every edge tree, recorded from the bash lint-status-parity.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 30) }

// TestLocked is LSP-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_nonempty_or_exit did, even beside a file that WARNs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class A {\n  @NiagaraProperty(\n    name = \"aInterval\"\n  )\n  @NiagaraProperty(\n    name = \"bInterval\"\n  )\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte(src), 0o644); err != nil {
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

// TestUnreadableFile pins a declared difference: a listed file the scan cannot read is "cannot scan" on
// stderr and exit 3, and the other files' rows still print. The bash lint let gawk print its fatal and
// exited 0 (fail-open).
func TestUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class Z {\n  @NiagaraProperty(\n    name = \"aInterval\"\n  )\n  @NiagaraProperty(\n    name = \"bInterval\"\n  )\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "Z.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	u := filepath.Join(dir, "A.java")
	if err := os.WriteFile(u, []byte(src), 0); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	wantOut := "WARN  lint-status-parity  Z  N=2 per-instance config slots but 0 status slots\n"
	wantErr := ID + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if code != 3 || o.String() != wantOut || e.String() != wantErr {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), wantOut, wantErr)
	}
}
