package wiringmap

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

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 63

// recordedKind is the @kind of a case testdata/record/record.sh wrote; any other @kind must be a truth-* case.
const recordedKind = "recorded from build-n4-module-kit/toolbelt/generate-wiring-map.sh at fa1201e6\n"

// TestGolden runs every testdata case (written by testdata/record/record.sh from the bash
// generate-wiring-map.sh, or hand-written as a truth-* case) in the golden format of
// internal/lint/rules/golden, with the section keys record.sh documents: "@kind", "@expect <file>" (a file
// the run must leave with that content), "<dir> (dir)", "<dir> (locked)", "<file> (unreadable)" and
// "<link> (symlink)". Each case runs with its tree as the working directory, as record.sh ran it.
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
			for _, d := range c.Dirs {
				materialize(t, dir, d+" (dir)", "")
			}
			expect := map[string]string{}
			for key, body := range c.Files {
				body = strings.ReplaceAll(body, "@DIR@", dir)
				switch {
				case key == "@kind":
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
			code := Main(args, cli.Stdio{Out: &o, Err: &e})
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

// materialize creates one tree section under dir; a mode-000 entry cannot block root, so such a case skips.
func materialize(t *testing.T, dir, key, body string) {
	t.Helper()
	name, kind, _ := strings.Cut(key, " (")
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if (kind == "locked)" || kind == "unreadable)") && os.Geteuid() == 0 {
		t.Skip("running as root: mode 000 does not block reads")
	}
	var err error
	switch kind {
	case "symlink)":
		err = os.Symlink(body, full)
	case "dir)":
		err = os.MkdirAll(full, 0o755)
	case "locked)":
		if err = os.MkdirAll(full, 0); err == nil {
			t.Cleanup(func() { _ = os.Chmod(full, 0o755) })
		}
	case "unreadable)":
		err = os.WriteFile(full, nil, 0)
	default:
		err = os.WriteFile(full, []byte(body), 0o644)
	}
	if err != nil {
		t.Fatal(err)
	}
}
