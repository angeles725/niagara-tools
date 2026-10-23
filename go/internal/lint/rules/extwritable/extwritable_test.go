package extwritable

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/ext-writable-shape.bats run on a tree and every edge tree, recorded from the bash
// lint (the empty-root and unreadable cases are table tests below).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 40) }

// run runs the rule alone, as its wrapper does, and returns the exit code, stdout and stderr with dir
// spelled @DIR@.
func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), strings.ReplaceAll(e.String(), dir, "@DIR@")
}

// write creates the files of a tree under a new temporary directory.
func write(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, text := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// room is a class whose line 2 declares an OPERATOR BStatusNumeric no action writes.
const room = "package a;\n@NiagaraProperty(name=\"sp\", type=\"BStatusNumeric\", flags=Flags.OPERATOR)\npublic class BA {}\n"

// TestEmptyRoot is EW11's empty directory (a golden case cannot hold one): the K20 ERROR row, exit 3.
func TestEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	code, out, stderr := run(t, dir, dir)
	want := "ERROR  ext-writable-shape  @DIR@  no Java sources found (K20: wrong path or empty scaffold)\n"
	if code != 3 || out != want || stderr != "" {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, out, stderr, want)
	}
}

// TestLocked is EW-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_or_exit did, even beside a file that WARNs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := write(t, map[string]string{"A.java": room})
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, stderr := run(t, dir, "--strict", dir)
	want := ID + ": cannot list every file under @DIR@: find: '@DIR@/locked': Permission denied\n"
	if code != 3 || out != "" || stderr != want {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, out, stderr, want)
	}
}
