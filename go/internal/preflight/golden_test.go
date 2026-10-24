package preflight

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 125

// recordedKind is the @kind of a case testdata/record/record.sh wrote; any other @kind must be a truth-* case.
const recordedKind = "recorded from build-n4-module-kit/toolbelt/preflight.sh at 5a4668f0\n"

// recordTools are the host tools record.sh links into each case's .tools (its TOOLS list).
var recordTools = []string{"bash", "sh", "env", "grep", "sed", "awk", "cut", "find", "paste", "sort", "uniq", "wc", "tr", "comm", "basename", "dirname", "head", "tail", "xargs", "df", "sleep", "cat", "printf", "mkdir", "ls", "readlink", "timeout", "rm", "seq"}

// TestGolden runs every testdata case (written by testdata/record/record.sh from the bash preflight.sh, or
// hand-edited from one into a truth-* case) in the golden format of internal/lint/rules/golden, with these
// section keys: "@kind" (the source; truth-* cases and only they start "truth:"), "@env" (the whole
// environment of the run: preflight and its commands see nothing else), "@expect <file>" (a file the run must
// leave with that content), "<dir>/" an empty directory, "<dir>/ (locked)" one with mode 000, "<file> (exec)"
// an executable file and "<link> (symlink)" a symlink to the body. Fake commands are scripts in the tree.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) != goldenCases {
		t.Fatalf("found %d golden cases, want exactly %d", len(paths), goldenCases)
	}
	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), ".txt")
		t.Run(name, func(t *testing.T) {
			c, err := golden.Read(p)
			if err != nil {
				t.Fatal(err)
			}
			kind, truth := c.Files["@kind"], strings.HasPrefix(name, "truth-")
			if truth != strings.HasPrefix(kind, "truth: ") || (!truth && kind != recordedKind) {
				t.Fatalf("@kind %q does not fit the case name (want %q, or truth: for a truth-* case)", kind, recordedKind)
			}
			dir := t.TempDir()
			env, expect := map[string]string{}, map[string]string{}
			for key, body := range c.Files {
				body = strings.ReplaceAll(body, "@DIR@", dir)
				switch {
				case key == "@kind":
				case key == "@env":
					for _, kv := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
						k, v, _ := strings.Cut(kv, "=")
						env[k] = v
					}
				case strings.HasPrefix(key, "@expect "):
					expect[strings.TrimPrefix(key, "@expect ")] = body
				default:
					materialize(t, dir, key, body)
				}
			}
			linkTools(t, dir)
			var o, e bytes.Buffer
			h := Host{Output: System.Output, Env: environ(env)}
			code := Command(h)(substDir(c.Args, dir), cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
			for f, want := range expect {
				if b, err := os.ReadFile(filepath.Join(dir, f)); err != nil || string(b) != want {
					t.Errorf("%s after the run = %q (%v), want %q", f, b, err, want)
				}
			}
		})
	}
}

func substDir(args []string, dir string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = strings.ReplaceAll(a, "@DIR@", dir)
	}
	return out
}

// environ is env as KEY=VALUE lines in key order (the Env of the case's commands).
func environ(env map[string]string) []string {
	out := []string{}
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// linkTools gives the case the <dir>/.tools record.sh gave it: the host's recordTools, so `#!/usr/bin/env bash`
// in a fake resolves through the case PATH alone.
func linkTools(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, ".tools"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, tool := range recordTools {
		if p, err := exec.LookPath(tool); err == nil {
			if err := os.Symlink(p, filepath.Join(dir, ".tools", tool)); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// materialize creates one tree section under dir. A locked directory cannot block root: the run fails
// unless PREFLIGHT_GOLDEN_ALLOW_ROOT_SKIP=1 says skipping those cases is intended.
func materialize(t *testing.T, dir, key, body string) {
	t.Helper()
	name, kind, _ := strings.Cut(key, " (")
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	var err error
	switch kind {
	case "symlink)":
		err = os.Symlink(body, full)
	case "locked)":
		if os.Geteuid() == 0 {
			if os.Getenv("PREFLIGHT_GOLDEN_ALLOW_ROOT_SKIP") != "1" {
				t.Fatal("running as root: mode 000 does not block reads; set PREFLIGHT_GOLDEN_ALLOW_ROOT_SKIP=1 to skip")
			}
			t.Skip("running as root: mode 000 does not block reads (PREFLIGHT_GOLDEN_ALLOW_ROOT_SKIP=1)")
		}
		if err = os.MkdirAll(full, 0o755); err == nil {
			err = os.Chmod(full, 0)
			t.Cleanup(func() { _ = os.Chmod(full, 0o755) })
		}
	case "exec)":
		err = os.WriteFile(full, []byte(body), 0o755)
	default:
		if strings.HasSuffix(name, "/") {
			err = os.MkdirAll(full, 0o755)
		} else {
			err = os.WriteFile(full, []byte(body), 0o644)
		}
	}
	if err != nil {
		t.Fatal(err)
	}
}
