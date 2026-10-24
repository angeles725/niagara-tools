package newretro

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 40

// recordedKind is the @kind of a case testdata/record/record.sh wrote; any other @kind must be a truth-* case.
const recordedKind = "recorded from build-n4-module-kit/toolbelt/new-retro.sh at 15e792d1\n"

// today is the date record.sh's fake `date` prints.
func today() string { return "2026-01-02" }

// TestGolden runs every testdata case (written by testdata/record/record.sh from the bash new-retro.sh, or
// hand-written as a truth-* case) in the golden format of internal/lint/rules/golden, with these section
// keys: "@kind" (the source; truth-* cases and only they start "truth:"), "@env" (the whole environment of
// the run, plus DefaultRootEnv = the case dir, the bash script's own ../), "@expect <file>" (every file the
// run must leave, with its content: a file not listed fails the case) and "<dir>/" an empty directory. The
// run's cwd is the case dir.
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
			env, expect := map[string]string{DefaultRootEnv: dir}, map[string]string{}
			for key, body := range c.Files {
				body = strings.ReplaceAll(body, "@DIR@", dir)
				switch {
				case key == "@kind":
				case key == "@env":
					for _, kv := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
						if k, v, ok := strings.Cut(kv, "="); ok {
							env[k] = v
						}
					}
				case strings.HasPrefix(key, "@expect "):
					expect[strings.TrimPrefix(key, "@expect ")] = body
				default:
					materialize(t, dir, key, body)
				}
			}
			args := make([]string, len(c.Args))
			for i, a := range c.Args {
				args[i] = strings.ReplaceAll(a, "@DIR@", dir)
			}
			t.Chdir(dir)
			var o, e bytes.Buffer
			code := Command(today)(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
			got := strings.ReplaceAll(fmt.Sprintf("exit %d\n%s", code, o.String()), dir, "@DIR@")
			if stderr := strings.ReplaceAll(e.String(), dir, "@DIR@"); got != c.Want || stderr != c.Stderr {
				t.Errorf("got\n%s-- stderr --\n%s\nwant\n%s-- stderr --\n%s", got, stderr, c.Want, c.Stderr)
			}
			if left := tree(t, dir); fmt.Sprint(left) != fmt.Sprint(expect) {
				t.Errorf("the run left\n%q\nwant\n%q", left, expect)
			}
		})
	}
}

// materialize creates one tree section: a path ending in "/" is an empty directory, any other a file.
func materialize(t *testing.T, dir, key, body string) {
	t.Helper()
	full := filepath.Join(dir, key)
	err := os.MkdirAll(filepath.Dir(full), 0o755)
	if strings.HasSuffix(key, "/") {
		err = os.MkdirAll(full, 0o755)
	} else if err == nil {
		err = os.WriteFile(full, []byte(body), 0o644)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// tree is every regular file under dir by its slash path relative to dir.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
