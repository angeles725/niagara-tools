// Package golden runs a rule's golden cases: testdata/<case>.txt files recorded from the bash lint the rule
// replaces, by pointing that lint's bats suite at a recording wrapper (go/README.md, Adding a lint rule).
//
// A case file holds "-- args --" (one argument per line), "-- want --" ("exit <n>" then the exact stdout),
// "-- stderr --" and one "-- <path> --" section per file of the tree; "-- <path> (noeol) --" is a file whose
// last line has no "\n". "@DIR@" stands for the tree's absolute path in every section.
package golden

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Case is one recorded run of the bash lint.
type Case struct {
	Args         []string
	Want, Stderr string
	Files        map[string]string
}

// Read parses one case file.
func Read(path string) (Case, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Case{}, err
	}
	c := Case{Files: map[string]string{}}
	name, body := "", ""
	flush := func() {
		switch {
		case name == "args":
			c.Args = strings.Split(strings.TrimSuffix(body, "\n"), "\n")
			if body == "" {
				c.Args = nil
			}
		case name == "want":
			c.Want = body
		case name == "stderr":
			c.Stderr = body
		case strings.HasSuffix(name, " (noeol)"):
			c.Files[strings.TrimSuffix(name, " (noeol)")] = strings.TrimSuffix(body, "\n")
		case name != "":
			c.Files[name] = body
		}
	}
	for _, ln := range strings.SplitAfter(string(data), "\n") {
		if strings.HasPrefix(ln, "-- ") && strings.HasSuffix(ln, " --\n") {
			flush()
			name, body = strings.TrimSuffix(strings.TrimPrefix(ln, "-- "), " --\n"), ""
			continue
		}
		body += ln
	}
	flush()
	return c, nil
}

// Run checks that spec, run alone as its toolbelt wrapper runs it, prints every case's recorded exit code,
// stdout and stderr byte for byte. It fails when testdata holds fewer than min cases.
func Run(t *testing.T, spec lint.Spec, min int) {
	t.Helper()
	command := lint.Command(spec)
	RunMain(t, func(args []string, s cli.Stdio) int { return command(append([]string{spec.ID()}, args...), s) }, min)
}

// RunMain is Run for a command that is not a lint rule: m gets each case's arguments as recorded.
func RunMain(t *testing.T, m cli.Main, min int) {
	t.Helper()
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < min {
		t.Fatalf("found %d golden cases, want at least %d", len(paths), min)
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			c, err := Read(p)
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
			code := m(args, cli.Stdio{Out: &o, Err: &e})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
		})
	}
}

// Unreadable checks the awk_lint_run failure contract of a rule that sets ContinueOnWalkError: under a root
// with an unreadable sub-directory and an unreadable file beside a.java (which holds src), the rule prints
// the walk env line, then the scan env line, prints the rows a.java alone gives (at least one) and exits 3.
func Unreadable(t *testing.T, spec lint.Spec, src string) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	run := func(dir string) (int, string, string) {
		var o, e bytes.Buffer
		code := lint.Command(spec)([]string{spec.ID(), dir}, cli.Stdio{Out: &o, Err: &e})
		return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), e.String()
	}
	clean, dir := t.TempDir(), t.TempDir()
	for _, d := range []string{clean, dir} {
		if err := os.MkdirAll(filepath.Join(d, "a"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "a", "A.java"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	u, locked := filepath.Join(dir, "a", "U.java"), filepath.Join(dir, "locked")
	if err := os.WriteFile(u, []byte(src), 0); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	_, rows, _ := run(clean)
	code, out, stderr := run(dir)
	want := spec.ID() + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n" +
		spec.ID() + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if rows == "" || code != 3 || out != rows || stderr != want {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, out, stderr, rows, want)
	}
}
