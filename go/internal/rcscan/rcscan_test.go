package rcscan

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/rc-scan.bats and tests/report-module.bats run of rc-scan.sh, and every edge tree,
// recorded from the awk script (go/README.md, Adding a lint rule, step 3): exit code, stdout and stderr byte
// for byte. truth-*.txt cases are hand-written after the port, one per truth fix.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 40 {
		t.Fatalf("found %d golden cases, want at least 40", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			t.Parallel()
			c, err := golden.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			for f, text := range c.Files {
				full := filepath.Join(dir, f)
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			var args []string
			for _, a := range c.Args {
				args = append(args, strings.ReplaceAll(a, "@DIR@", dir))
			}
			var o, e bytes.Buffer
			code := Main(args, cli.Stdio{Out: &o, Err: &e})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
		})
	}
}

// TestFailClosed: an unreadable sub-directory stops the scan before any row (RS-locked), and an unreadable
// file, where gawk printed its fatal error and the script went on, is an env error too.
func TestFailClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "rc"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "rc", "a.js"), []byte("var a = 'station:|x';\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	run := func() (int, string, string) {
		var o, e bytes.Buffer
		code := Main([]string{dir}, cli.Stdio{Out: &o, Err: &e})
		return code, o.String(), e.String()
	}
	if code, out, err := run(); code != 3 || out != "" || err != "rc-scan: cannot list every file under "+dir+": find: '"+locked+"': Permission denied\n" {
		t.Errorf("locked dir: exit %d out %q err %q", code, out, err)
	}
	if err := os.Chmod(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	u := filepath.Join(dir, "rc", "u.js")
	if err := os.WriteFile(u, nil, 0); err != nil {
		t.Fatal(err)
	}
	if code, out, err := run(); code != 3 || out != "" || err != "rc-scan: cannot scan "+u+": Permission denied\n" {
		t.Errorf("unreadable file: exit %d out %q err %q", code, out, err)
	}
}
