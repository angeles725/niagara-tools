package lint

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// fakeRule reports every raw line that contains its marker.
type fakeRule struct {
	id, marker string
	sev        Severity
}

func (r fakeRule) ID() string         { return r.id }
func (r fakeRule) Severity() Severity { return r.sev }
func (r fakeRule) Check(f *javasrc.File) []Finding {
	var out []Finding
	for i, ln := range f.Raw {
		if strings.Contains(ln, r.marker) {
			out = append(out, Finding{Line: i + 1, Msg: "saw: " + ln})
		}
	}
	return out
}

func javaSpec(r Rule, mode Mode) Spec {
	return Spec{Rule: r, Mode: mode, Usage: "usage: " + r.ID() + ".sh <src-root>", Select: Selection{Names: []string{"*.java"}}}
}

// tree writes files (path -> text) under a fresh temp dir and returns it.
func tree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, text := range files {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func stdio() (cli.Stdio, *bytes.Buffer, *bytes.Buffer) {
	var o, e bytes.Buffer
	return cli.Stdio{Out: &o, Err: &e}, &o, &e
}

func skipIfRoot(t *testing.T) {
	t.Helper()
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
}

func TestRow(t *testing.T) {
	f := Finding{Rule: "lint-x", Severity: Warn, Path: "M/src/A.java", Line: 7, Msg: "null-Context write: a.set(b, c, null);"}
	if got, want := f.Row(), "WARN  lint-x  M/src/A.java:7  null-Context write: a.set(b, c, null);"; got != want {
		t.Errorf("Row() = %q, want %q", got, want)
	}
	f.Severity = Fail
	if got := f.Row(); !strings.HasPrefix(got, "FAIL  lint-x  ") {
		t.Errorf("Fail row = %q", got)
	}
}

func TestParse(t *testing.T) {
	const name, usage = "lint-x", "usage: lint-x.sh <src-root>"
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mode   Mode
		args   []string
		strict bool
		roots  []string
		errMsg string
	}{
		{Raw, nil, false, nil, usage + "\n"},
		{Raw, []string{dir}, false, []string{dir}, ""},
		{Raw, []string{dir, "extra", "--strict"}, false, []string{dir}, ""},
		{Raw, []string{"--strict", dir}, false, nil, name + ": not a directory: --strict\n"},
		{Raw, []string{"-h"}, false, nil, name + ": not a directory: -h\n"},
		{Raw, []string{file}, false, nil, name + ": not a directory: " + file + "\n"},
		{One, []string{"--strict", dir}, true, []string{dir}, ""},
		{One, []string{dir, dir}, false, nil, usage + "\n"},
		{One, []string{"--", "--strict"}, false, nil, name + ": not a directory: --strict\n"},
		{One, []string{"-x", dir}, false, nil, name + ": unknown flag: -x\n" + usage + "\n"},
		{First, []string{dir, "missing"}, false, []string{dir}, ""},
		{First, []string{"--strict"}, true, nil, usage + "\n"},
		{Each, []string{dir, dir}, false, []string{dir, dir}, ""},
		{Each, []string{dir, "missing"}, false, nil, name + ": not a directory: missing\n"},
	} {
		got, err := Parse(tc.mode, name, usage, tc.args)
		if tc.errMsg != "" {
			if err == nil || err.Error() != tc.errMsg {
				t.Errorf("Parse(%v, %q) error = %v, want %q", tc.mode, tc.args, err, tc.errMsg)
			}
			continue
		}
		if err != nil || got.Strict != tc.strict || !reflect.DeepEqual(got.Roots, tc.roots) {
			t.Errorf("Parse(%v, %q) = %+v, %v; want strict=%v roots=%q", tc.mode, tc.args, got, err, tc.strict, tc.roots)
		}
	}
}

func TestWalkOrderAndSelection(t *testing.T) {
	root := tree(t, map[string]string{
		"b/Z.java": "", "a/B.java": "", "a/A.java": "", "A.java": "", "a/x.txt": "",
		".git/G.java": "", "a/.hidden/H.java": "", ".dot.java": "", "build/O.java": "",
	})
	if err := os.Symlink(filepath.Join(root, "a"), filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "A.java"), filepath.Join(root, "L.java")); err != nil {
		t.Fatal(err)
	}
	sel := Selection{Names: []string{"*.java"}, Prune: []string{"build"}}
	got, err := Walk(root, sel)
	want := []string{root + "/.dot.java", root + "/A.java", root + "/a/A.java", root + "/a/B.java", root + "/b/Z.java"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %q, %v\nwant %q", got, err, want)
	}
	// find joins a root that ends in "/" without adding another one.
	if got, _ := Walk(root+"/", sel); len(got) == 0 || got[0] != root+"/.dot.java" {
		t.Errorf("Walk(root/) first = %q", got)
	}
	if got, _ := Walk(root+"//", sel); len(got) == 0 || got[0] != root+"//.dot.java" {
		t.Errorf("Walk(root//) first = %q", got)
	}
	// A symlinked root is not followed (find -P lists nothing under it).
	if got, err := Walk(filepath.Join(root, "link"), sel); err != nil || len(got) != 0 {
		t.Errorf("Walk(symlink root) = %q, %v", got, err)
	}
}

func TestWalkFailClosed(t *testing.T) {
	skipIfRoot(t)
	root := tree(t, map[string]string{"locked/A.java": "", "ok/B.java": ""})
	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	got, err := Walk(root, Selection{Names: []string{"*.java"}})
	if err == nil || err.Error() != "find: '"+locked+"': Permission denied" {
		t.Errorf("Walk error = %v", err)
	}
	if want := []string{root + "/ok/B.java"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Walk kept %q, want %q (the readable part is still listed)", got, want)
	}
}

func TestRunExitCodes(t *testing.T) {
	root := tree(t, map[string]string{"src/A.java": "x\nMARK one\n", "src/B.java": "clean\n", "src/C.txt": "MARK\n"})
	spec := javaSpec(fakeRule{"lint-x", "MARK", Warn}, One)
	s, o, e := stdio()
	if code := Run([]Spec{spec}, "lint-x", Args{Roots: []string{root}}, s); code != 0 {
		t.Errorf("WARN without --strict exit %d, want 0", code)
	}
	if want := "WARN  lint-x  " + root + "/src/A.java:2  saw: MARK one\n"; o.String() != want || e.Len() != 0 {
		t.Errorf("out %q err %q, want %q", o, e, want)
	}
	s, _, _ = stdio()
	if code := Run([]Spec{spec}, "lint-x", Args{Strict: true, Roots: []string{root}}, s); code != 1 {
		t.Errorf("WARN with --strict exit %d, want 1", code)
	}
	clean := tree(t, map[string]string{"B.java": "clean\n"})
	s, o, _ = stdio()
	if code := Run([]Spec{spec}, "lint-x", Args{Strict: true, Roots: []string{clean}}, s); code != 0 || o.Len() != 0 {
		t.Errorf("clean --strict exit %d out %q, want 0 and nothing", code, o)
	}
}

func TestRunUnreadableFile(t *testing.T) {
	skipIfRoot(t)
	root := tree(t, map[string]string{"A.java": "MARK\n", "B.java": "MARK\n"})
	a := filepath.Join(root, "A.java")
	if err := os.Chmod(a, 0); err != nil {
		t.Fatal(err)
	}
	s, o, e := stdio()
	code := Run([]Spec{javaSpec(fakeRule{"lint-x", "MARK", Warn}, One)}, "lint-x", Args{Roots: []string{root}}, s)
	if code != 3 {
		t.Errorf("exit %d, want 3", code)
	}
	if want := "lint-x: cannot scan " + a + ": awk: fatal: cannot open file `" + a + "' for reading: Permission denied\n"; e.String() != want {
		t.Errorf("err %q\nwant %q", e, want)
	}
	if !strings.Contains(o.String(), "B.java:1") {
		t.Errorf("the scan stopped at the unreadable file: out %q", o)
	}
}

func TestRunUnlistableDir(t *testing.T) {
	skipIfRoot(t)
	root := tree(t, map[string]string{"locked/A.java": "", "B.java": "MARK\n"})
	locked := filepath.Join(root, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	s, o, e := stdio()
	code := Run([]Spec{javaSpec(fakeRule{"lint-x", "MARK", Warn}, One)}, "lint-x", Args{Roots: []string{root}}, s)
	if want := "lint-x: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"; code != 3 || e.String() != want {
		t.Errorf("exit %d err %q, want 3 and %q", code, e, want)
	}
	if !strings.Contains(o.String(), "B.java:1") {
		t.Errorf("the listed files were not scanned: out %q", o)
	}
}

func TestRunRuleOrderAndSelection(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK OTHER\n", "build/B.java": "MARK OTHER\n"})
	first := javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each)
	second := javaSpec(fakeRule{"lint-b", "OTHER", Fail}, Each)
	second.Select.Prune = []string{"build"}
	s, o, _ := stdio()
	Run([]Spec{first, second}, "n4lint", Args{Roots: []string{root}}, s)
	want := "WARN  lint-a  " + root + "/A.java:1  saw: MARK OTHER\n" +
		"FAIL  lint-b  " + root + "/A.java:1  saw: MARK OTHER\n" +
		"WARN  lint-a  " + root + "/build/B.java:1  saw: MARK OTHER\n"
	if o.String() != want {
		t.Errorf("out\n%s\nwant\n%s", o, want)
	}
}

func TestCommand(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Warn}, Raw), javaSpec(fakeRule{"lint-b", "MARK", Warn}, One))
	run := func(args ...string) (int, string, string) {
		s, o, e := stdio()
		return main(args, s), o.String(), e.String()
	}
	if code, _, e := run(); code != 3 || !strings.HasPrefix(e, "usage: n4lint ") {
		t.Errorf("no args: %d %q", code, e)
	}
	if code, _, e := run("lint-zz", root); code != 3 || !strings.HasPrefix(e, "n4lint: unknown rule: lint-zz\nusage: n4lint ") {
		t.Errorf("unknown rule: %d %q", code, e)
	}
	if code, o, _ := run("-h"); code != 0 || !strings.Contains(o, "lint-a\n") || !strings.Contains(o, "lint-b\n") {
		t.Errorf("-h: %d %q", code, o)
	}
	// One rule: its own mode, name and usage (Raw takes --strict as the root).
	if code, _, e := run("lint-a", "--strict", root); code != 3 || e != "lint-a: not a directory: --strict\n" {
		t.Errorf("raw rule: %d %q", code, e)
	}
	if code, _, e := run("lint-a"); code != 3 || e != "usage: lint-a.sh <src-root>\n" {
		t.Errorf("raw rule usage: %d %q", code, e)
	}
	if code, o, _ := run("lint-b", "--strict", root); code != 1 || !strings.Contains(o, "WARN  lint-b  ") {
		t.Errorf("one rule --strict: %d %q", code, o)
	}
	// Several rules or --all: the n4lint contract ([--strict] [--] <root>...).
	if code, o, _ := run("lint-a,lint-b", "--strict", root, root); code != 1 || strings.Count(o, "\n") != 4 {
		t.Errorf("rule list: %d %q", code, o)
	}
	if code, o, _ := run("--all", root); code != 0 || !strings.Contains(o, "lint-a") || !strings.Contains(o, "lint-b") {
		t.Errorf("--all: %d %q", code, o)
	}
	if code, _, e := run("--all", "--bogus", root); code != 3 || !strings.HasPrefix(e, "n4lint: unknown flag: --bogus\n") {
		t.Errorf("--all unknown flag: %d %q", code, e)
	}
}
