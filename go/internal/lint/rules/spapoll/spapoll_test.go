package spapoll

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

// TestGolden: every tests/lint-spa-poll-no-recovery.bats run (SPR-awkfail and SPR-finderr aside:
// TestUnreadable pins them) and every edge tree, recorded from the bash lint-spa-poll-no-recovery.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 33) }

// TestUnreadable is SPR-awkfail and SPR-finderr under awk_lint_run's contract (ContinueOnWalkError): an
// unreadable sub-directory and an unreadable file beside a.js print the walk and scan env lines, a.js's row
// still prints, exit 3. (golden.Unreadable writes .java files, which this rule does not select.)
func TestUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "function poll() { a().catch(e => 0); }\nsetInterval(poll, 1);\n"
	for name, mode := range map[string]os.FileMode{"a.js": 0o644, "u.js": 0} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), mode); err != nil {
			t.Fatal(err)
		}
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	u := filepath.Join(dir, "u.js")
	wantErr := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		ID + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if code != 3 || !strings.HasPrefix(o.String(), "WARN  "+ID+"  "+filepath.Join(dir, "a.js")+":2  ") || e.String() != wantErr {
		t.Errorf("exit %d out %q err %q, want 3, the a.js row and %q", code, o.String(), e.String(), wantErr)
	}
}
