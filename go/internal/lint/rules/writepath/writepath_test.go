package writepath

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-write-path.bats and tests/write-path-drift.bats run, the report-module
// fixtures and the edge trees, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 78) }

// TestZippedBog: a saved station config.bog is file.xml zipped; it gives the rows of the same bog read
// as plain XML (the edge-bog-module-root tree, which has bog-linked FAIL rows).
func TestZippedBog(t *testing.T) {
	c, err := golden.Read("testdata/edge-bog-module-root.txt")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for p, text := range c.Files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var zipped bytes.Buffer
	zw := zip.NewWriter(&zipped)
	w, _ := zw.Create("file.xml")
	if _, err := w.Write([]byte(c.Files["bog/config.bog"])); err != nil || zw.Close() != nil {
		t.Fatal("cannot zip the bog")
	}
	zbog := filepath.Join(dir, "bog", "zipped.bog")
	if err := os.WriteFile(zbog, zipped.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(bog string) (int, string) {
		var o, e bytes.Buffer
		code := lint.Command(Spec())([]string{"lint-write-path", filepath.Join(dir, "bog/ColdRoomPan"), "--bog", bog}, cli.Stdio{Out: &o, Err: &e})
		return code, o.String() + e.String()
	}
	pc, plain := run(filepath.Join(dir, "bog", "config.bog"))
	zc, got := run(zbog)
	if zc != pc || got != plain || !strings.Contains(got, "slot panelSlot") {
		t.Errorf("zipped: exit %d\n%s\nplain: exit %d\n%s", zc, got, pc, plain)
	}
}

func write(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for p, text := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{"lint-write-path"}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

const opSrc = "@NiagaraProperty(name=\"lockedKnob\", flags=Flags.OPERATOR)\n"

// TestUnreadable (WP-finderr, WP-finderr-matrix): a directory either walk cannot read is the bash lint's
// ERROR row, exit 3; an unreadable Java file, which its awk and grep skipped in silence (the slot passed
// as covered), fails closed too: "lint-write-path: cannot scan <path>: <reason>" on stderr, exit 3.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "docs/write-path-matrix.md": "| `setpoint` | a |\n",
		"M/src/a/B.java": opSrc, "M/src/b/C.java": opSrc, "O/x/A.java": "class A {}\n"})
	m, locked, file, other := dir+"/M", dir+"/M/src/a", dir+"/M/src/b/C.java", dir+"/O/x"
	for _, tc := range []struct {
		lock     string
		out, err string
	}{
		{locked, "lint-write-path  ERROR  " + m + "  cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n", ""},
		{other, "lint-write-path  ERROR  " + m + "  cannot list every file under " + dir + ": find: '" + other + "': Permission denied\n", ""},
		{file, "", "lint-write-path: cannot scan " + file + ": Permission denied\n"},
	} {
		if err := os.Chmod(tc.lock, 0); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := run(m)
		_ = os.Chmod(tc.lock, 0o755)
		if code != 3 || out != tc.out || errOut != tc.err {
			t.Errorf("%s locked: exit %d out %q err %q, want 3, %q and %q", tc.lock, code, out, errOut, tc.out, tc.err)
		}
	}
}

// TestSymlinkedRoot (WP-symroot, issue #329): a symlinked module root lists its profiles like find -H and
// prints the directory's rows spelled from the link.
func TestSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/docs/write-path-matrix.md": "| `setpoint` | a |\n",
		"M/M-rt/src/B.java": opSrc})
	link := filepath.Join(dir, "L")
	if err := os.Symlink(filepath.Join(dir, "M"), link); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(filepath.Join(dir, "M"))
	lcode, lout, _ := run(link)
	if want := strings.ReplaceAll(out, filepath.Join(dir, "M"), link); lcode != 1 || code != 1 || lout != want || !strings.Contains(lout, "lockedKnob") {
		t.Errorf("link: exit %d out\n%s\nwant 1 and\n%s", lcode, lout, want)
	}
}
