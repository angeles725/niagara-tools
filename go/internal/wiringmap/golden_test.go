package wiringmap

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// goldenCases is the exact number of testdata cases: a lost or an extra case fails the run.
const goldenCases = 65

// recordedKind is the @kind of a case testdata/record/record.sh wrote; any other @kind must be a truth-* case.
const recordedKind = "recorded from build-n4-module-kit/toolbelt/generate-wiring-map.sh at 15e792d1\n"

// TestGolden runs every testdata case (written by testdata/record/record.sh from the bash
// generate-wiring-map.sh, or hand-written as a truth-* case) in the golden format of
// internal/lint/rules/golden, with the section keys record.sh documents: "@kind", "@tools" (ignored here),
// "@expect <file>" (a file the run creates or changes, with its content after the run), "<dir> (dir)",
// "<dir> (locked)", "<file> (unreadable)" and "<link> (symlink)". Each case runs with its tree as the working
// directory, as record.sh ran it. After the run the whole tree must be the case tree with the @expect files
// applied: a missing, extra or changed file fails, so a run that must write nothing is pinned too.
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
			want := map[string]string{} // the tree after the run: path -> entry (see entry)
			for key, body := range c.Files {
				body = strings.ReplaceAll(body, "@DIR@", dir)
				switch name, kind, _ := strings.Cut(key, " ("); {
				case strings.HasPrefix(key, "@expect "):
					want[strings.TrimPrefix(key, "@expect ")] = "file " + body
				case strings.HasPrefix(key, "@"):
				default:
					materialize(t, dir, key, body)
					if _, set := want[name]; !set && kind != "dir)" && kind != "locked)" {
						want[name] = map[string]string{"": "file " + body, "symlink)": "symlink " + body, "unreadable)": "unreadable"}[kind]
					}
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
			if got := tree(t, dir); !reflect.DeepEqual(got, want) {
				t.Errorf("tree after the run\n got %q\nwant %q", got, want)
			}
		})
	}
}

// tree is every file and symlink under dir after a run, keyed by its path relative to dir: "file <bytes>",
// "symlink <target>" or "unreadable" (a mode-000 file). A directory the walk cannot enter holds nothing.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	got := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		rel, _ := filepath.Rel(dir, p)
		switch {
		case err != nil && d != nil && d.IsDir():
			return fs.SkipDir
		case err != nil:
			return err
		case d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(p)
			got[rel] = "symlink " + target
			return err
		case d.Type().IsRegular():
			if b, err := os.ReadFile(p); err == nil {
				got[rel] = "file " + string(b)
			} else if errors.Is(err, fs.ErrPermission) {
				got[rel] = "unreadable"
			} else {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// materialize creates one tree section under dir. A mode-000 entry cannot block root: the run fails unless
// WIRINGMAP_GOLDEN_ALLOW_ROOT_SKIP=1 says skipping those cases is intended.
func materialize(t *testing.T, dir, key, body string) {
	t.Helper()
	name, kind, _ := strings.Cut(key, " (")
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if (kind == "locked)" || kind == "unreadable)") && os.Geteuid() == 0 {
		if os.Getenv("WIRINGMAP_GOLDEN_ALLOW_ROOT_SKIP") != "1" {
			t.Fatal("running as root: mode 000 does not block reads; set WIRINGMAP_GOLDEN_ALLOW_ROOT_SKIP=1 to skip")
		}
		t.Skip("running as root: mode 000 does not block reads (WIRINGMAP_GOLDEN_ALLOW_ROOT_SKIP=1)")
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
