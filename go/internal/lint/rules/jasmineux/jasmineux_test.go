package jasmineux

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-jasmine-ux.bats run (JUX-finderr aside: TestDirs pins it) and every edge
// tree, recorded from the bash lint-jasmine-ux.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 19) }

// run runs the rule on root and returns its exit code, stdout and stderr.
func run(root string) (int, string, string) {
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, root}, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

// TestDirs pins the shapes a golden tree cannot hold (empty directories) and JUX-finderr: an empty src/rc
// still applies, an empty spec directory is no specs, an unreadable spec sub-directory is exit 3 with no row.
func TestDirs(t *testing.T) {
	warn := func(root string) string {
		return "WARN  " + ID + "  " + root + "  ux ships src/rc but no srcTest/rc/spec/*.js\n"
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "src", "rc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run(root); code != 0 || out != warn(root) || errs != "" {
		t.Errorf("empty src/rc: exit %d out %q err %q", code, out, errs)
	}
	spec := filepath.Join(root, "srcTest", "rc", "spec")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run(root); code != 0 || out != warn(root) {
		t.Errorf("empty spec dir: exit %d out %q", code, out)
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	locked := filepath.Join(spec, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	want := ID + ": cannot list every file under " + root + "/srcTest/rc/spec: find: '" + locked + "': Permission denied\n"
	if code, out, errs := run(root); code != 3 || out != "" || errs != want {
		t.Errorf("locked: exit %d out %q err %q, want 3, no row and %q", code, out, errs, want)
	}
}
