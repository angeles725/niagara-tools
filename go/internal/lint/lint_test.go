package lint

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
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
func (r fakeRule) Check(f *javasrc.File) ([]Finding, error) {
	var out []Finding
	for i, ln := range f.Raw {
		if strings.Contains(ln, r.marker) {
			out = append(out, Finding{Line: i + 1, Msg: "saw: " + ln})
		}
	}
	return out, nil
}

// mixedRule reports lines holding WARN (severity left unset) or FAIL (severity set by the rule) and
// cannot analyze a file that holds BROKEN.
type mixedRule struct{}

func (mixedRule) ID() string         { return "lint-m" }
func (mixedRule) Severity() Severity { return Warn }
func (mixedRule) Check(f *javasrc.File) ([]Finding, error) {
	var out []Finding
	for i, ln := range f.Raw {
		switch {
		case strings.Contains(ln, "BROKEN"):
			return out, errors.New("unbalanced braces")
		case strings.Contains(ln, "FAIL"):
			out = append(out, Finding{Severity: Fail, Line: i + 1, Msg: "saw: " + ln})
		case strings.Contains(ln, "WARN"):
			out = append(out, Finding{Line: i + 1, Msg: "saw: " + ln})
		}
	}
	return out, nil
}

// treeFake is a TreeRule: it records the paths it is given, reports a file-level finding (sub-check
// "sub") for each file holding MARK and a PASS row for the others, and fails when broken is set.
type treeFake struct {
	id     string
	seen   *[]string
	broken bool
}

func (r treeFake) ID() string         { return r.id }
func (r treeFake) Severity() Severity { return Fail }
func (r treeFake) CheckTree(files []*javasrc.File) ([]Finding, error) {
	var out []Finding
	for _, f := range files {
		if r.seen != nil {
			*r.seen = append(*r.seen, f.Path)
		}
		if strings.Contains(strings.Join(f.Raw, "\n"), "MARK") {
			out = append(out, Finding{Check: "sub", Path: f.Path, Msg: "tree saw MARK"})
		} else {
			out = append(out, Finding{Check: "sub", Severity: Pass, Path: f.Path, Msg: "clean"})
		}
	}
	if r.broken {
		return out, errors.New("cannot scan " + files[0].Path + ": broken")
	}
	return out, nil
}

// bothRule implements Rule and TreeRule, which registration refuses.
type bothRule struct{ fakeRule }

func (bothRule) CheckTree([]*javasrc.File) ([]Finding, error) { return nil, nil }

func treeSpec(r TreeRule) Spec {
	return Spec{Tree: r, Mode: One, Usage: "usage: " + r.ID() + ".sh <src-root>", Select: Selection{Names: []string{"*.java"}}}
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

func TestRowColumns(t *testing.T) {
	f := Finding{Rule: "lint-t", Check: "timer-ticket", Severity: Pass, Path: "M/A.java", Msg: "timer cancelled in stopped()"}
	if got, want := f.Row(), "PASS  timer-ticket  M/A.java: timer cancelled in stopped()"; got != want {
		t.Errorf("file-level PASS row = %q, want %q", got, want)
	}
	f.Line, f.Severity = 3, Warn
	if got, want := f.Row(), "WARN  timer-ticket  M/A.java:3  timer cancelled in stopped()"; got != want {
		t.Errorf("line row with sub-check = %q, want %q", got, want)
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
		{Exact, nil, false, nil, usage + "\n"},
		{Exact, []string{dir}, false, []string{dir}, ""},
		{Exact, []string{dir, dir}, false, nil, usage + "\n"},
		{Exact, []string{"-h"}, false, nil, name + ": not a directory: -h\n"},
		{Exact, []string{"--strict"}, false, nil, name + ": not a directory: --strict\n"},
		{Exact, []string{"--strict", dir}, false, nil, usage + "\n"},
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
		"x-worktrees/wt/W.java": "", "a/y-worktrees/V.java": "",
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
	// A symlinked root is followed like find -H (issue #329), spelled as given; links below it are not.
	// Mutation: WALK-symroot -- os.Lstat instead of os.Stat on the root lists nothing under a symlinked root.
	outer := t.TempDir()
	if err := os.Symlink(root, filepath.Join(outer, "mod")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(outer, "mod")
	want = []string{link + "/.dot.java", link + "/A.java", link + "/a/A.java", link + "/a/B.java", link + "/b/Z.java"}
	if got, err := Walk(link, sel); err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Walk(symlink root) = %q, %v\nwant %q", got, err, want)
	}
	if got, err := Walk(filepath.Join(root, "link"), sel); err != nil || !reflect.DeepEqual(got, []string{root + "/link/A.java", root + "/link/B.java"}) {
		t.Errorf("Walk(inner symlink as root) = %q, %v", got, err)
	}
	// A dangling root symlink is an incomplete walk, never an empty clean list.
	dangling := filepath.Join(outer, "gone")
	if err := os.Symlink(filepath.Join(outer, "nothing"), dangling); err != nil {
		t.Fatal(err)
	}
	if got, err := Walk(dangling, sel); err == nil || err.Error() != "find: '"+dangling+"': No such file or directory" || len(got) != 0 {
		t.Errorf("Walk(dangling root) = %q, %v", got, err)
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

// lockedTree is tree with dir made unreadable (restored at cleanup); it returns the root and dir's path.
func lockedTree(t *testing.T, files map[string]string, dir string) (string, string) {
	t.Helper()
	skipIfRoot(t)
	root := tree(t, files)
	locked := filepath.Join(root, dir)
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	return root, locked
}

// TestRunUnlistableDir: by default an incomplete walk stops the rule, as scan_files_or_exit does: the
// env line only, no rows from the files that were listed, no later root, exit 3.
func TestRunUnlistableDir(t *testing.T) {
	root, locked := lockedTree(t, map[string]string{"locked/A.java": "", "B.java": "MARK\n"}, "locked")
	later := tree(t, map[string]string{"C.java": "MARK\n"})
	s, o, e := stdio()
	code := Run([]Spec{javaSpec(fakeRule{"lint-x", "MARK", Warn}, Each)}, "lint-x", Args{Roots: []string{root, later}}, s)
	if want := "lint-x: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"; code != 3 || e.String() != want || o.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, o, e, want)
	}
}

// TestRunUnlistableDirContinue: a ContinueOnWalkError rule scans the files that were listed, as
// awk_lint_run does: the env line, then the rows, then the next root, exit 3.
func TestRunUnlistableDirContinue(t *testing.T) {
	root, locked := lockedTree(t, map[string]string{"locked/A.java": "", "B.java": "MARK\n"}, "locked")
	later := tree(t, map[string]string{"C.java": "MARK\n"})
	spec := javaSpec(fakeRule{"lint-x", "MARK", Warn}, Each)
	spec.ContinueOnWalkError = true
	s, o, e := stdio()
	code := Run([]Spec{spec}, "lint-x", Args{Roots: []string{root, later}}, s)
	wantOut := "WARN  lint-x  " + root + "/B.java:1  saw: MARK\nWARN  lint-x  " + later + "/C.java:1  saw: MARK\n"
	wantErr := "lint-x: cannot list every file under " + root + ": find: '" + locked + "': Permission denied\n"
	if code != 3 || o.String() != wantOut || e.String() != wantErr {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o, e, wantOut, wantErr)
	}
}

// TestRunUnlistableDirMixed: in one run a stopping rule prints nothing while a continuing rule scans.
func TestRunUnlistableDirMixed(t *testing.T) {
	root, _ := lockedTree(t, map[string]string{"locked/A.java": "", "B.java": "MARK\n"}, "locked")
	cont := javaSpec(fakeRule{"lint-c", "MARK", Warn}, Each)
	cont.ContinueOnWalkError = true
	s, o, _ := stdio()
	code := Run([]Spec{javaSpec(fakeRule{"lint-s", "MARK", Warn}, Each), cont}, "n4lint", Args{Roots: []string{root}}, s)
	if want := "WARN  lint-c  " + root + "/B.java:1  saw: MARK\n"; code != 3 || o.String() != want {
		t.Errorf("exit %d out %q, want 3 and %q", code, o, want)
	}
}

// TestWalkErrorText pins the bytes GNU find 4.9 prints for a directory it cannot read under LC_ALL=C
// (awk-lint.sh): always quoted; \' and \\; C escapes for \a \b \f \n \r \t \v; 3-digit octal for any
// other control byte and every byte from 0x80. Every row was measured on real directories.
func TestWalkErrorText(t *testing.T) {
	for path, want := range map[string]string{
		"src/locked":           `'src/locked'`,
		"src/lo\nck":           `'src/lo\nck'`,
		"src/lo\tck":           `'src/lo\tck'`,
		"src/q'b\\s\x01":       `'src/q\'b\\s\001'`,
		"src/\xc3\xa9\x7f\xff": `'src/\303\251\177\377'`,
		"src/a\ab\bc\fd\re\vf": `'src/a\ab\bc\fd\re\vf'`,
		"src/sp ace\x1b~":      `'src/sp ace\033~'`,
	} {
		err := &WalkError{Path: path, Err: syscall.EACCES}
		if got, w := err.Error(), "find: "+want+": Permission denied"; got != w {
			t.Errorf("WalkError(%q) = %q, want %q", path, got, w)
		}
	}
}

func TestRunRuleOrderAndSelection(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK OTHER\n", "build/B.java": "MARK OTHER\n"})
	first := javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each)
	second := javaSpec(fakeRule{"lint-b", "OTHER", Fail}, Each)
	second.Select.Prune = []string{"build"}
	s, o, _ := stdio()
	Run([]Spec{first, second}, "n4lint", Args{Roots: []string{root}}, s)
	// Each rule's rows stay contiguous, in spec order.
	want := "WARN  lint-a  " + root + "/A.java:1  saw: MARK OTHER\n" +
		"WARN  lint-a  " + root + "/build/B.java:1  saw: MARK OTHER\n" +
		"FAIL  lint-b  " + root + "/A.java:1  saw: MARK OTHER\n"
	if o.String() != want {
		t.Errorf("out\n%s\nwant\n%s", o, want)
	}
}

// TestRunExitPolicy: any FAIL row exits 1 with or without --strict, --strict also makes a WARN row exit 1,
// and an env error (here a file the rule cannot analyze) exits 3 even beside a FAIL row.
func TestRunExitPolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		text   string
		strict bool
		want   int
	}{
		{"FAIL alone", "FAIL a\n", false, 1},
		{"WARN alone", "WARN a\n", false, 0},
		{"WARN with --strict", "WARN a\n", true, 1},
		{"env error with FAIL", "FAIL a\nBROKEN\n", false, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := tree(t, map[string]string{"A.java": tc.text})
			s, _, _ := stdio()
			if code := Run([]Spec{javaSpec(mixedRule{}, One)}, "lint-m", Args{Strict: tc.strict, Roots: []string{root}}, s); code != tc.want {
				t.Errorf("exit %d, want %d", code, tc.want)
			}
		})
	}
}

func TestRunPerFindingSeverity(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "FAIL one\nWARN two\n"})
	s, o, _ := stdio()
	if code := Run([]Spec{javaSpec(mixedRule{}, One)}, "lint-m", Args{Roots: []string{root}}, s); code != 1 {
		t.Errorf("exit %d, want 1 (a FAIL row)", code)
	}
	want := "FAIL  lint-m  " + root + "/A.java:1  saw: FAIL one\n" +
		"WARN  lint-m  " + root + "/A.java:2  saw: WARN two\n"
	if o.String() != want {
		t.Errorf("out\n%s\nwant\n%s", o, want)
	}
}

func TestRunCheckErrorFailsClosed(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "WARN a\nBROKEN\n", "B.java": "WARN b\n"})
	s, o, e := stdio()
	code := Run([]Spec{javaSpec(mixedRule{}, One)}, "lint-m", Args{Roots: []string{root}}, s)
	if code != 3 {
		t.Errorf("exit %d, want 3", code)
	}
	a := root + "/A.java"
	if want := "lint-m: cannot scan " + a + ": lint-m: unbalanced braces\n"; e.String() != want {
		t.Errorf("err %q\nwant %q", e, want)
	}
	// The findings made before the error and the other file's rows are still printed.
	want := "WARN  lint-m  " + a + ":1  saw: WARN a\n" + "WARN  lint-m  " + root + "/B.java:1  saw: WARN b\n"
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

func TestRunTreeRuleMixedOrder(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n", "b/B.java": "MARK\n", "C.java": "clean\n", "build/D.java": "MARK\n"})
	var seen []string
	tr := treeSpec(treeFake{id: "lint-t", seen: &seen})
	tr.Select.Prune = []string{"build"}
	specs := []Spec{javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each), tr, javaSpec(fakeRule{"lint-b", "clean", Warn}, Each)}
	s, o, e := stdio()
	code := Run(specs, "n4lint", Args{Roots: []string{root}}, s)
	want := "WARN  lint-a  " + root + "/A.java:1  saw: MARK\n" +
		"WARN  lint-a  " + root + "/b/B.java:1  saw: MARK\n" +
		"WARN  lint-a  " + root + "/build/D.java:1  saw: MARK\n" +
		"FAIL  sub  " + root + "/A.java: tree saw MARK\n" +
		"PASS  sub  " + root + "/C.java: clean\n" +
		"FAIL  sub  " + root + "/b/B.java: tree saw MARK\n" +
		"WARN  lint-b  " + root + "/C.java:1  saw: clean\n"
	if o.String() != want || e.Len() != 0 || code != 1 {
		t.Errorf("exit %d err %q out\n%s\nwant\n%s", code, e, o, want)
	}
	if w := []string{root + "/A.java", root + "/C.java", root + "/b/B.java"}; !reflect.DeepEqual(seen, w) {
		t.Errorf("tree rule saw %q, want the ordered selected files %q", seen, w)
	}
}

func TestRunPassNeverExits(t *testing.T) {
	root := tree(t, map[string]string{"C.java": "clean\n"})
	s, o, _ := stdio()
	if code := Run([]Spec{treeSpec(treeFake{id: "lint-t"})}, "lint-t", Args{Strict: true, Roots: []string{root}}, s); code != 0 {
		t.Errorf("PASS-only --strict exit %d, want 0 (out %q)", code, o)
	}
}

func TestRunTreeRuleError(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	s, o, e := stdio()
	code := Run([]Spec{treeSpec(treeFake{id: "lint-t", broken: true})}, "lint-t", Args{Roots: []string{root}}, s)
	if want := "FAIL  sub  " + root + "/A.java: tree saw MARK\n"; code != 3 || o.String() != want {
		t.Errorf("exit %d out %q, want 3 and %q", code, o, want)
	}
	if want := "lint-t: cannot scan " + root + "/A.java: broken\n"; e.String() != want {
		t.Errorf("err %q, want %q", e, want)
	}
}

func TestCommandRegistration(t *testing.T) {
	for name, sp := range map[string]Spec{
		"neither":         {Usage: "u"},
		"both fields":     {Rule: fakeRule{id: "lint-a"}, Tree: treeFake{id: "lint-a"}, Usage: "u"},
		"Rule is a tree":  {Rule: bothRule{fakeRule{id: "lint-a"}}, Usage: "u"},
		"Tree is a check": {Tree: bothRule{fakeRule{id: "lint-a"}}, Usage: "u"},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: Command accepted the spec", name)
				}
			}()
			Command(sp)
		}()
	}
	Command(javaSpec(fakeRule{id: "lint-a"}, One), treeSpec(treeFake{id: "lint-t"})) // a valid mix registers
}

func TestCommandUsageExit(t *testing.T) {
	def := treeSpec(treeFake{id: "lint-d"})
	two := treeSpec(treeFake{id: "lint-u"})
	two.UsageExit = 2
	main := Command(def, two)
	s, _, e := stdio()
	if code := main([]string{"lint-d"}, s); code != 3 || e.String() != "usage: lint-d.sh <src-root>\n" {
		t.Errorf("default usage exit %d %q, want 3", code, e)
	}
	s, _, e = stdio()
	if code := main([]string{"lint-u", "a", "b"}, s); code != 2 || e.String() != "usage: lint-u.sh <src-root>\n" {
		t.Errorf("UsageExit 2: exit %d %q", code, e)
	}
	s, _, _ = stdio()
	if code := main([]string{"lint-u", "-x", "a"}, s); code != 2 {
		t.Errorf("UsageExit 2, unknown flag: exit %d, want 2", code)
	}
	s, _, _ = stdio()
	if code := main([]string{"lint-u", "missing-dir"}, s); code != 3 {
		t.Errorf("UsageExit 2, not a directory: exit %d, want 3 (an env error, not usage)", code)
	}
	s, _, _ = stdio()
	if code := main([]string{"lint-u,lint-d"}, s); code != 3 {
		t.Errorf("rule list usage: exit %d, want 3 (the n4lint contract)", code)
	}
}

func TestCommandRuleOrder(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n", "B.java": "MARK\n"})
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Warn}, One), treeSpec(treeFake{id: "lint-t"}))
	perFile := "WARN  lint-a  " + root + "/A.java:1  saw: MARK\n" + "WARN  lint-a  " + root + "/B.java:1  saw: MARK\n"
	treeRows := "FAIL  sub  " + root + "/A.java: tree saw MARK\n" + "FAIL  sub  " + root + "/B.java: tree saw MARK\n"
	for args, want := range map[string]string{"lint-t,lint-a": treeRows + perFile, "--all": perFile + treeRows} {
		s, o, _ := stdio()
		if code := main([]string{args, root}, s); code != 1 || o.String() != want {
			t.Errorf("%s: exit %d out\n%s\nwant\n%s", args, code, o, want)
		}
	}
}

func TestParseExactUsageError(t *testing.T) {
	for _, args := range [][]string{nil, {"a", "b"}, {"--", "a"}} {
		_, err := Parse(Exact, "lint-x", "usage: lint-x.sh <java-root>", args)
		var ue *UsageError
		if !errors.As(err, &ue) {
			t.Errorf("Parse(Exact, %q) error %v, want a *UsageError", args, err)
		}
	}
	main := Command(Spec{Tree: treeFake{id: "lint-e"}, Mode: Exact, Usage: "usage: lint-e.sh <java-root>", UsageExit: 2,
		Select: Selection{Names: []string{"*.java"}}})
	s, _, e := stdio()
	if code := main([]string{"lint-e", "--strict", "x"}, s); code != 2 || e.String() != "usage: lint-e.sh <java-root>\n" {
		t.Errorf("Exact two args: exit %d %q, want 2 and the usage line", code, e)
	}
	s, _, e = stdio()
	if code := main([]string{"lint-e", "-h"}, s); code != 3 || e.String() != "lint-e: not a directory: -h\n" {
		t.Errorf("Exact -h: exit %d %q, want 3 (a root, not a flag)", code, e)
	}
}
