package structure

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

// TestGolden: every tests/lint-structure.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 42) }

func run(args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{"lint-structure"}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
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

// TestSymlinkedRoot (LS-symroot, issue #329): a symlinked module root is walked like find -H and prints
// the directory's rows spelled from the link.
func TestSymlinkedRoot(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "M/gradle.properties": "niagara_home=C:\\N\n", "M/Foo-rt/module-include.xml": "<types/>\n"})
	link := filepath.Join(dir, "L")
	if err := os.Symlink(filepath.Join(dir, "M"), link); err != nil {
		t.Fatal(err)
	}
	code, out, _ := run(filepath.Join(dir, "M"))
	lcode, lout, _ := run(link)
	if want := strings.ReplaceAll(out, filepath.Join(dir, "M"), link); lcode != code || lout != want || !strings.Contains(lout, "L10") {
		t.Errorf("link: exit %d out\n%s\nwant %d and\n%s", lcode, lout, code, want)
	}
}

// TestUnreadable: an unreadable root build.gradle.kts is a visible L15 WARN (LS15-unreadable, exit 0);
// any other file or directory the lint cannot read fails closed (exit 3), where the bash grep and find
// went on as if it held no match.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	write(t, dir, map[string]string{".git/HEAD": "x\n", "settings.gradle.kts": "// root\n",
		"build.gradle.kts": `plugins { id("com.tridium.niagara-signing") }` + "\n", "M/A-rt/module-include.xml": "<types/>\n",
		"M/A-rt/module.palette": "<p n=\"A\"/>\n", "M/A-rt/src/A.java": "class A {}\n"})
	build, java, src := filepath.Join(dir, "build.gradle.kts"), filepath.Join(dir, "M/A-rt/src/A.java"), filepath.Join(dir, "M/A-rt/src")
	for _, p := range []string{build, java} {
		if err := os.Chmod(p, 0); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { _ = os.Chmod(src, 0o755) })
	l15 := "WARN  lint-structure  " + build + "  L15: could not scan the root for the signing block (awk exit 2: awk: fatal: cannot open file `" +
		build + "' for reading: Permission denied) — signing block unchecked\n"
	code, out, errs := run(filepath.Join(dir, "M"))
	if want := "lint-structure: cannot scan " + java + ": Permission denied\n"; code != 3 || out != l15 || errs != want {
		t.Errorf("unreadable file: exit %d out %q err %q, want 3, %q and %q", code, out, errs, l15, want)
	}
	if err := os.Chmod(src, 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs = run(filepath.Join(dir, "M"))
	// the first walk to meet it is L10's over the whole module root
	if want := "lint-structure: cannot list every file under " + filepath.Join(dir, "M") + ": find: '" + src + "': Permission denied\n"; code != 3 || out != l15 || errs != want {
		t.Errorf("unlistable dir: exit %d out %q err %q, want 3, %q and %q", code, out, errs, l15, want)
	}
}
