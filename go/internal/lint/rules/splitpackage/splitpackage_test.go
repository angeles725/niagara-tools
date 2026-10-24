package splitpackage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/split-package-check.bats run on a tree the format can hold and every edge tree, recorded
// from the bash (origin/main 53bfe0c).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 28) }

// TestUnreadable: a directory the module search cannot enter, one under a module's src/ and a source file that
// cannot be read are one stderr line each (texts recorded from the bash), the rows still print and the exit is 3.
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	p := filepath.Join(t.TempDir(), "proj")
	write := func(rel, text string) {
		full := filepath.Join(p, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range []string{"A/A-rt", "B/B-rt"} {
		write(m+"/build.gradle.kts", "")
		write(m+"/src/x/U.java", "package s;\n")
	}
	write("B/B-rt/src/x/V.java", "package s;\n")
	lock := func(rel string) {
		full := filepath.Join(p, rel)
		if err := os.MkdirAll(full, 0o755); err != nil && !os.IsExist(err) {
			t.Fatal(err)
		}
		if err := os.Chmod(full, 0); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(full, 0o755) })
	}
	lock("locked")
	lock("A/A-rt/src/lk")
	v := filepath.Join(p, "B/B-rt/src/x/V.java")
	if err := os.Chmod(v, 0); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, p}, cli.Stdio{Out: &o, Err: &e})
	row := "WARN  split-package-check  s  declared in 2 modules: A/A-rt (A/A-rt/src/x/U.java), B/B-rt (B/B-rt/src/x/U.java) -- first-dependency-wins shadows one copy; one package belongs to one module (types/issues-and-gotchas.md §D4c)\n"
	errs := ID + ": cannot list every file under " + p + ": find: '" + p + "/locked': Permission denied\n" +
		ID + ": cannot list every file under " + p + "/A/A-rt/src: find: '" + p + "/A/A-rt/src/lk': Permission denied\n" +
		ID + ": cannot scan " + v + ": awk: fatal: cannot open file `" + v + "' for reading: Permission denied\n"
	if code != 3 || o.String() != row || e.String() != errs {
		t.Errorf("exit %d\nout %q\nerr %q\nwant 3\n%q\n%q", code, o.String(), e.String(), row, errs)
	}
}
