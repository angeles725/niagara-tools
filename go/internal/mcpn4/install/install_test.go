package install

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kitcli "github.com/angeles725/niagara-tools/go/internal/cli"
)

func writeFile(t *testing.T, path, text string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, d := range des {
		names = append(names, d.Name())
	}
	return names
}

func TestInstallCopiesTheBinaryExecutableIntoAFreshPrefix(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "build", "mcp-n4")
	writeFile(t, src, "new binary", 0o700)
	prefix := filepath.Join(root, "home", ".local", "bin")
	dst, changed, err := Install(src, prefix)
	if err != nil || !changed || dst != filepath.Join(prefix, "mcp-n4") {
		t.Fatalf("Install = %q %v %v, want %s changed", dst, changed, err, filepath.Join(prefix, "mcp-n4"))
	}
	st, err := os.Stat(dst)
	if err != nil || st.Mode().Perm() != 0o755 || readFile(t, dst) != "new binary" {
		t.Fatalf("installed %v %v, want mode 0755 and the source bytes", st, err)
	}
	if got := entries(t, prefix); len(got) != 1 {
		t.Fatalf("prefix holds %v, want only mcp-n4 (no temp file left)", got)
	}
}

func TestInstallReplacesAnExistingTargetOnlyByRename(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "mcp-n4")
	writeFile(t, src, "new binary", 0o755)
	prefix := filepath.Join(root, "bin")
	dst := filepath.Join(prefix, "mcp-n4")
	writeFile(t, dst, "old binary", 0o755)
	// A hard link to the old file: an in-place write would change it, a rename leaves it.
	held := filepath.Join(root, "held")
	if err := os.Link(dst, held); err != nil {
		t.Fatal(err)
	}
	if _, changed, err := Install(src, prefix); err != nil || !changed {
		t.Fatalf("Install = %v %v, want changed", changed, err)
	}
	if readFile(t, dst) != "new binary" || readFile(t, held) != "old binary" {
		t.Fatalf("target %q, old inode %q: the old file was written in place", readFile(t, dst), readFile(t, held))
	}
}

func TestInstallFailureLeavesTheOldBinaryAndNoTempFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "mcp-n4")
	writeFile(t, src, "new binary", 0o755)
	prefix := filepath.Join(root, "bin")
	dst := filepath.Join(prefix, "mcp-n4")
	writeFile(t, dst, "old binary", 0o755)
	old := rename
	rename = func(string, string) error { return errors.New("injected rename failure") }
	t.Cleanup(func() { rename = old })
	if _, _, err := Install(src, prefix); err == nil || !strings.Contains(err.Error(), "injected") {
		t.Fatalf("Install err = %v, want the rename failure", err)
	}
	if readFile(t, dst) != "old binary" {
		t.Fatal("a failed install changed the old binary")
	}
	if got := entries(t, prefix); len(got) != 1 {
		t.Fatalf("prefix holds %v after a failure, want only the old mcp-n4", got)
	}
}

func TestInstallIsANoOpWhenTheTargetAlreadyHoldsTheSameBytes(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "mcp-n4")
	writeFile(t, src, "same", 0o755)
	prefix := filepath.Join(root, "bin")
	writeFile(t, filepath.Join(prefix, "mcp-n4"), "same", 0o755)
	if _, changed, err := Install(src, prefix); err != nil || changed {
		t.Fatalf("Install = %v %v, want unchanged", changed, err)
	}
}

func TestMainDefaultsThePrefixToHomeLocalBin(t *testing.T) {
	home := t.TempDir()
	src := filepath.Join(t.TempDir(), "mcp-n4")
	writeFile(t, src, "bin", 0o755)
	old := executable
	executable = func() (string, error) { return src, nil }
	t.Cleanup(func() { executable = old })
	var out, errb bytes.Buffer
	s := kitcli.Stdio{Out: &out, Err: &errb, Getenv: func(n string) string {
		if n == "HOME" {
			return home
		}
		return ""
	}}
	if code := Main(nil, s); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errb.String())
	}
	want := filepath.Join(home, ".local", "bin", "mcp-n4")
	if readFile(t, want) != "bin" || !strings.Contains(out.String(), want) {
		t.Fatalf("stdout %q, want the binary installed and named at %s", out.String(), want)
	}
	if code := Main([]string{"--bogus"}, s); code != 2 {
		t.Fatalf("an unknown flag exits %d, want 2", code)
	}
}

func TestInstallRemovesOnlyItsOwnStaleTemporaryFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src", "mcp-n4")
	writeFile(t, src, "bin", 0o755)
	prefix := filepath.Join(root, "bin")
	old := time.Now().Add(-2 * time.Hour)
	for _, n := range []string{".mcp-n4-stale.tmp", ".mcp-n4-young.tmp", ".other-stale.tmp"} {
		writeFile(t, filepath.Join(prefix, n), "x", 0o600)
	}
	for _, n := range []string{".mcp-n4-stale.tmp", ".other-stale.tmp"} {
		if err := os.Chtimes(filepath.Join(prefix, n), old, old); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := Install(src, prefix); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(entries(t, prefix), ","); got != ".mcp-n4-young.tmp,.other-stale.tmp,mcp-n4" {
		t.Fatalf("prefix holds %s, want the stale mcp-n4 temp removed and the others kept", got)
	}
}

func TestMainSaysItReplacedASymlink(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	src := filepath.Join(t.TempDir(), "mcp-n4")
	writeFile(t, src, "bin", 0o755)
	prefix := filepath.Join(home, ".local", "bin")
	writeFile(t, filepath.Join(home, "wt", "mcp-n4"), "bin", 0o755)
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(home, "wt", "mcp-n4"), filepath.Join(prefix, "mcp-n4")); err != nil {
		t.Fatal(err)
	}
	old := executable
	executable = func() (string, error) { return src, nil }
	t.Cleanup(func() { executable = old })
	var out, errb bytes.Buffer
	if code := Main(nil, kitcli.Stdio{Out: &out, Err: &errb, Getenv: func(string) string { return home }}); code != 0 ||
		!strings.Contains(out.String(), "replaced the symlink to "+filepath.Join(home, "wt", "mcp-n4")) {
		t.Fatalf("exit %d, stdout %q, want the replaced symlink named", code, out.String())
	}
}
