package lint

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"

	"fmt"
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
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
	id            string
	seen          *[]string
	broken, empty bool
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
	switch {
	case r.empty:
		return out, ErrEmptyRoot
	case r.broken:
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
		{Trailing, nil, false, nil, usage + "\n"},
		{Trailing, []string{dir}, false, []string{dir}, ""},
		{Trailing, []string{dir, "--strict", "--strict"}, true, []string{dir}, ""},
		{Trailing, []string{dir, "--strict", "-x"}, false, nil, name + ": unknown option: -x\n"},
		{Trailing, []string{dir, dir}, false, nil, name + ": unknown option: " + dir + "\n"},
		{Trailing, []string{"--strict", dir}, false, nil, name + ": unknown option: " + dir + "\n"}, // options are checked before the root
		{Trailing, []string{"--strict"}, false, nil, name + ": not a directory: --strict\n"},
		{Trailing, []string{"--", dir}, false, nil, name + ": unknown option: " + dir + "\n"},
		{Trailing, []string{"-h"}, false, nil, name + ": not a directory: -h\n"},
		{Leading, nil, false, nil, usage + "\n"},
		{Leading, []string{"--strict"}, true, nil, usage + "\n"},
		{Leading, []string{"--strict", dir, "missing"}, true, []string{dir}, ""},
		{Leading, []string{dir, "--strict"}, false, []string{dir}, ""},
		{Leading, []string{"--strict", "--strict", dir}, true, nil, name + ": not a directory: --strict\n"},
		{Leading, []string{"--", dir}, false, nil, name + ": not a directory: --\n"},
		{Leading, []string{"-h", dir}, false, nil, name + ": not a directory: -h\n"},
		{Leading, []string{""}, false, nil, name + ": not a directory: \n"},
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

// TestWalkPathSelection pins the find tests rc-scan.sh passed to scan_files: -path '*/rc/*' on the file,
// -not -name '*.min.js', and --prune-path '*/rc/ext' (a pruned directory is never entered, so an unreadable
// one is no walk error); the root itself is never pruned.
func TestWalkPathSelection(t *testing.T) {
	root := tree(t, map[string]string{
		"rc/a.js": "", "rc/a.min.js": "", "rc/ext/x.js": "", "rc/js/ext/y.js": "", "rc/sub/rc/ext/z.js": "",
		"src/b.js": "", "rcx/c.js": "", "m/rc/d.css": "", "rc/ext2/e.js": "",
	})
	if err := os.Mkdir(filepath.Join(root, "rc", "ext", "locked"), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(root, "rc", "ext", "locked"), 0o755) })
	sel := Selection{Names: []string{"*.js", "*.css"}, Except: []string{"*.min.js"}, Paths: []string{"*/rc/*"}, PrunePaths: []string{"*/rc/ext"}}
	got, err := Walk(root, sel)
	want := []string{root + "/m/rc/d.css", root + "/rc/a.js", root + "/rc/ext2/e.js", root + "/rc/js/ext/y.js"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("Walk = %q, %v\nwant %q", got, err, want)
	}
	for _, p := range append(want, root+"/rc/ext/x.js", root+"/rc/a.min.js", root+"/src/b.js") {
		if keep := sel.keeps(root, p); keep != slices.Contains(want, p) {
			t.Errorf("keeps(%q) = %v", p, keep)
		}
	}
	// The root is never pruned: under <root>/rc/ext every file is listed.
	if got, err := Walk(root+"/rc/ext", sel); err == nil || len(got) != 1 || got[0] != root+"/rc/ext/x.js" {
		t.Errorf("Walk(rc/ext root) = %q, %v; want x.js and the locked directory's error", got, err)
	}
	for _, tc := range []struct {
		pat, s string
		want   bool
	}{{"*/rc/*", "a/rc/b/c", true}, {"*/rc/*", "rc/b", false}, {"*/rc/ext", "a/rc/ext", true},
		{"*/rc/ext", "a/rc/ext/b", false}, {"a?c", "a/c", true}, {"", "", true}, {"*", "", true}} {
		if got := findPath(tc.pat, tc.s); got != tc.want {
			t.Errorf("findPath(%q, %q) = %v", tc.pat, tc.s, got)
		}
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
		"Root and Rule":   {Rule: fakeRule{id: "lint-a"}, Root: rootFake{}, Usage: "u"},
		"Root is a check": {Root: rootAndCheck{fakeRule{id: "lint-a"}}, Usage: "u"},
		"Rule is a root":  {Rule: rootAndCheck{fakeRule{id: "lint-a"}}, Usage: "u"},
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
	Command(javaSpec(fakeRule{id: "lint-a"}, One), treeSpec(treeFake{id: "lint-t"}), rootSpec(rootFake{})) // a valid mix registers
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

// TestTreeRuleEmptyRoot: a TreeRule whose selected files hold nothing it checks (lint-duplicate-helper: no
// -rt/-ux/-wb src/ under the root) returns ErrEmptyRoot: the EmptyError row instead of its findings, the
// rule stops (no later root) and the exit is 3, as for a RootRule.
func TestTreeRuleEmptyRoot(t *testing.T) {
	a := tree(t, map[string]string{"A.java": "MARK\n"})
	b := tree(t, map[string]string{"B.java": "MARK\n"})
	sp := treeSpec(treeFake{id: "lint-t", empty: true})
	sp.Mode = Each
	sp.EmptyError = EmptyRow{Msg: "nothing to compare"}
	s, o, e := stdio()
	if code := Command(sp)([]string{"lint-t", a, b}, s); code != 3 || o.String() != "ERROR  lint-t  "+a+"  nothing to compare\n" || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 3 and one ERROR row", code, o, e)
	}
}

// TestModuleRootLeftOutOfAll: --all runs on source roots, so it leaves out a rule whose roots are module
// roots (Spec.ModuleRoot), as it leaves out a RootRule; by name the rule runs.
func TestModuleRootLeftOutOfAll(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	mod := treeSpec(treeFake{id: "lint-m"})
	mod.ModuleRoot = true
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Warn}, One), mod)
	s, o, _ := stdio()
	if code := main([]string{"--all", root}, s); code != 0 || o.String() != "WARN  lint-a  "+root+"/A.java:1  saw: MARK\n" {
		t.Errorf("--all: exit %d out\n%s\nwant the lint-a row only", code, o)
	}
	s, o, _ = stdio()
	if code := main([]string{"lint-m", root}, s); code != 1 || o.String() != "FAIL  sub  "+root+"/A.java: tree saw MARK\n" {
		t.Errorf("by name: exit %d out\n%s\nwant the lint-m row", code, o)
	}
}

// TestShortScanError: Spec.ShortScanError words an unreadable file as awk_lint_each did ("<lint>: cannot scan
// <path>: <reason>"), not as awk_lint_run's awk error; the other files are still scanned and the exit is 3.
func TestShortScanError(t *testing.T) {
	skipIfRoot(t)
	root := tree(t, map[string]string{"A.java": "MARK\n", "U.java": "x\n"})
	u := filepath.Join(root, "U.java")
	if err := os.Chmod(u, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(u, 0o644) })
	sp := javaSpec(fakeRule{"lint-s", "MARK", Fail}, Raw)
	sp.ShortScanError = true
	s, o, e := stdio()
	if code := Command(sp)([]string{"lint-s", root}, s); code != 3 || o.String() != "FAIL  lint-s  "+root+"/A.java:1  saw: MARK\n" ||
		e.String() != "lint-s: cannot scan "+u+": Permission denied\n" {
		t.Errorf("exit %d out %q err %q", code, o, e)
	}
}

func TestEmptyOK(t *testing.T) {
	sp := javaSpec(fakeRule{"lint-e", "MARK", Fail}, Raw)
	sp.EmptyOK = true
	empty := tree(t, map[string]string{"README": "x\n"})
	s, o, e := stdio()
	if code := Command(sp)([]string{"lint-e", empty}, s); code != 0 || o.Len() != 0 || e.Len() != 0 {
		t.Errorf("empty: exit %d out %q err %q, want 0 and nothing", code, o, e)
	}
	full := tree(t, map[string]string{"A.java": "MARK\n"})
	s, o, _ = stdio()
	if code := Command(sp)([]string{"lint-e", full}, s); code != 1 || o.String() != "FAIL  lint-e  "+full+"/A.java:1  saw: MARK\n" {
		t.Errorf("full: exit %d out %q", code, o)
	}
}

// flagRule reports its configured n once per file (Spec.IntFlags).
type flagRule struct{ n int }

func (flagRule) ID() string { return "lint-f" }

func (flagRule) Severity() Severity { return Warn }

func (r flagRule) Check(*javasrc.File) ([]Finding, error) {
	return []Finding{{Line: 1, Msg: fmt.Sprintf("n=%d", r.n)}}, nil
}

// TestCommandIntFlags: Spec.IntFlags are read among the leading flags in any order with --strict (the last
// one wins), fail with the "needs a non-negative integer" line on a bad value, leave an unknown flag to
// Parse, stop at "--", and do not exist for a rule list.
func TestCommandIntFlags(t *testing.T) {
	sp := javaSpec(flagRule{80}, First)
	sp.IntFlags = []IntFlag{{Name: "--max", Default: 80}}
	sp.Configure = func(v map[string]int) Rule { return flagRule{v["--max"]} }
	main := Command(sp, javaSpec(fakeRule{"lint-a", "MARK", Warn}, First))
	root := tree(t, map[string]string{"A.java": "x\n"})
	row := func(n string) string { return "WARN  lint-f  " + root + "/A.java:1  n=" + n + "\n" }
	bad := func(f string) string {
		return "lint-f: " + f + " needs a non-negative integer\nusage: lint-f.sh <src-root>\n"
	}
	for _, tc := range []struct {
		name     string
		args     []string
		code     int
		out, err string
	}{
		{"default", []string{"lint-f", root}, 0, row("80"), ""},
		{"given", []string{"lint-f", "--max", "5", root}, 0, row("5"), ""},
		{"with strict", []string{"lint-f", "--max", "007", "--strict", root}, 1, row("7"), ""},
		{"last wins", []string{"lint-f", "--max", "5", "--max", "0", root}, 0, row("0"), ""},
		{"missing value", []string{"lint-f", "--max"}, 3, "", bad("--max")},
		{"not a count", []string{"lint-f", "--max", "-1", root}, 3, "", bad("--max")},
		{"empty value", []string{"lint-f", "--max", "", root}, 3, "", bad("--max")},
		{"unknown flag first", []string{"lint-f", "--bogus", "--max", "x", root}, 3, "",
			"lint-f: unknown flag: --bogus\nusage: lint-f.sh <src-root>\n"},
		{"after --", []string{"lint-f", "--", "--max", root}, 3, "", "lint-f: not a directory: --max\n"},
		{"rule list", []string{"lint-f,lint-a", "--max", "5", root}, 3, "",
			"n4lint: unknown flag: --max\n" + usage + "\n"},
	} {
		s, o, e := stdio()
		if code := main(tc.args, s); code != tc.code || o.String() != tc.out || e.String() != tc.err {
			t.Errorf("%s: exit %d out %q err %q, want %d %q %q", tc.name, code, o, e, tc.code, tc.out, tc.err)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("IntFlags without Configure: no panic")
		}
	}()
	noConf := javaSpec(flagRule{1}, First)
	noConf.IntFlags = sp.IntFlags
	Command(noConf)
}

// TestParseValued: Mode Valued is `[<option> <value>]... <root>` (lint-link-target-flags): the option and the
// root in any order, the last value wins, a missing value, any other "-" argument, a second root or no root is
// the usage line alone, and the root must be a directory.
func TestParseValued(t *testing.T) {
	dir := t.TempDir()
	const use = "usage: lint-v.sh [--map <file>] <src>"
	for _, tc := range []struct {
		name string
		args []string
		opt  string
		err  string
	}{
		{"option first", []string{"--map", "m.md", dir}, "m.md", ""},
		{"option after the root", []string{dir, "--map", "m.md"}, "m.md", ""},
		{"last value wins", []string{"--map", "a", "--map", "b", dir}, "b", ""},
		{"no option", []string{dir}, "", ""},
		{"missing value", []string{dir, "--map"}, "", use + "\n"},
		{"another flag", []string{"--strict", dir}, "", use + "\n"},
		{"-h", []string{"-h"}, "", use + "\n"},
		{"two roots", []string{dir, dir}, "", use + "\n"},
		{"no root", []string{"--map", "m"}, "", use + "\n"},
		{"not a directory", []string{"--map", "m", dir + "/nope"}, "", "lint-v: not a directory: " + dir + "/nope\n"},
	} {
		a, err := ParseOption(Valued, "lint-v", use, "--map", tc.args)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != tc.err || (err == nil && (a.Option != tc.opt || len(a.Roots) != 1 || a.Roots[0] != dir)) {
			t.Errorf("%s: args %+v err %q, want option %q root %q err %q", tc.name, a, got, tc.opt, dir, tc.err)
		}
	}
}

// configFake is a Rule that is also a Configurer: it fails Configure when the option names "bad".
type configFake struct {
	fakeRule
	seen *string
}

func (r configFake) Configure(a Args) error {
	*r.seen = a.Option
	if a.Option == "bad" {
		return errors.New("lint-c: --map file not found: bad\n")
	}
	return nil
}

// TestConfigure: a rule run alone that implements Configurer gets the parsed Args after the root check
// and before the walk; its error is printed verbatim on stderr and exits 3 with no row.
func TestConfigure(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	var seen string
	sp := Spec{Rule: configFake{fakeRule{"lint-c", "MARK", Fail}, &seen}, Mode: Valued, Option: "--map",
		Usage: "usage: lint-c.sh [--map <file>] <src>", Select: Selection{Names: []string{"*.java"}}}
	s, o, e := stdio()
	if code := Command(sp)([]string{"lint-c", "--map", "good", root}, s); code != 1 || seen != "good" || o.String() != "FAIL  lint-c  "+root+"/A.java:1  saw: MARK\n" || e.Len() != 0 {
		t.Errorf("good: exit %d seen %q out %q err %q", code, seen, o, e)
	}
	s, o, e = stdio()
	if code := Command(sp)([]string{"lint-c", "--map", "bad", root}, s); code != 3 || o.Len() != 0 || e.String() != "lint-c: --map file not found: bad\n" {
		t.Errorf("bad: exit %d out %q err %q", code, o, e)
	}
	s, _, e = stdio()
	if code := Command(sp)([]string{"lint-c", "--map", "bad", root + "/nope"}, s); code != 3 || e.String() != "lint-c: not a directory: "+root+"/nope\n" {
		t.Errorf("root first: exit %d err %q, want the root error before Configure", code, e)
	}
}

// TestCheckFirst: Spec.CheckFirst prints a finding as `<check>  <SEV>  <path>:<line>  <msg>` (lint-wb-threading's
// rows, which report-module relays as a "swap" member); the empty-root ERROR row keeps the standard shape.
func TestCheckFirst(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	sp := javaSpec(fakeRule{"lint-c", "MARK", Warn}, One)
	sp.CheckFirst = true
	s, o, _ := stdio()
	if code := Command(sp)([]string{"lint-c", root}, s); code != 0 || o.String() != "lint-c  WARN  "+root+"/A.java:1  saw: MARK\n" {
		t.Errorf("row: exit %d out %q", code, o)
	}
	empty := tree(t, map[string]string{"README": "x\n"})
	s, o, _ = stdio()
	if code := Command(sp)([]string{"lint-c", empty}, s); code != 3 || !strings.HasPrefix(o.String(), "ERROR  lint-c  "+empty+"  ") {
		t.Errorf("empty: exit %d out %q", code, o)
	}
}

// TestParseAnywhere: Mode Anywhere is lint-wb-threading's loop: --strict anywhere, another "-" argument is
// "<name>: unknown option: <arg>" and a second root "<name>: unexpected argument: <arg>" (exit 3, no usage
// line), no argument or no root is the usage line.
func TestParseAnywhere(t *testing.T) {
	dir := t.TempDir()
	const use = "usage: lint-w.sh [--strict] <src>"
	for _, tc := range []struct {
		args   []string
		strict bool
		err    string
		usage  bool
	}{
		{[]string{dir}, false, "", false},
		{[]string{dir, "--strict"}, true, "", false},
		{[]string{"--strict", dir, "--strict"}, true, "", false},
		{nil, false, use + "\n", true},
		{[]string{"--strict"}, false, use + "\n", true},
		{[]string{"-h", dir}, false, "lint-w: unknown option: -h\n", false},
		{[]string{dir, "--"}, false, "lint-w: unknown option: --\n", false},
		{[]string{dir, dir, "-x"}, false, "lint-w: unexpected argument: " + dir + "\n", false},
		{[]string{dir + "/nope"}, false, "lint-w: not a directory: " + dir + "/nope\n", false},
	} {
		a, err := Parse(Anywhere, "lint-w", use, tc.args)
		got := ""
		if err != nil {
			got = err.Error()
		}
		var ue *UsageError
		if got != tc.err || errors.As(err, &ue) != tc.usage || (err == nil && (a.Strict != tc.strict || len(a.Roots) != 1 || a.Roots[0] != dir)) {
			t.Errorf("%q: args %+v err %q (usage %v), want strict %v err %q (usage %v)", tc.args, a, got, errors.As(err, &ue), tc.strict, tc.err, tc.usage)
		}
	}
}

// TestParseValuedRepeat: Mode Valued with repeatable options (Spec.Options: every value kept, in order, in
// Args.Values) and --strict accepted (Spec.ValuedStrict), as lint-inline-site-literal takes them; without
// ValuedStrict --strict stays a usage error (lint-link-target-flags).
func TestParseValuedRepeat(t *testing.T) {
	dir := t.TempDir()
	const use = "usage: lint-r.sh [--strict] [--table <p>]... [--pattern <re>]... <root>"
	vo := ValueOptions{Repeat: []string{"--table", "--pattern"}, Strict: true}
	a, err := ParseValued(Valued, "lint-r", use, vo, []string{"--table", "a", dir, "--strict", "--pattern", "R[0-9]", "--table", "b"})
	if err != nil || !a.Strict || len(a.Roots) != 1 || a.Roots[0] != dir ||
		!reflect.DeepEqual(a.Values, map[string][]string{"--table": {"a", "b"}, "--pattern": {"R[0-9]"}}) {
		t.Errorf("repeat: %+v %v", a, err)
	}
	for _, args := range [][]string{{dir, "--table"}, {"--other", dir}, {dir, dir}} {
		if _, err := ParseValued(Valued, "lint-r", use, vo, args); err == nil || err.Error() != use+"\n" {
			t.Errorf("%q: err %v, want the usage line", args, err)
		}
	}
	if _, err := ParseValued(Valued, "lint-r", use, ValueOptions{Repeat: []string{"--table"}}, []string{"--strict", dir}); err == nil {
		t.Errorf("--strict without ValueOptions.Strict parsed")
	}
	// Command passes the spec's options: Values reach Configure.
	var seen []string
	sp := Spec{Rule: valuesFake{fakeRule{"lint-r", "MARK", Warn}, &seen}, Mode: Valued, Options: []string{"--table"}, ValuedStrict: true,
		Usage: use, Select: Selection{Names: []string{"*.java"}}}
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	s, _, _ := stdio()
	if code := Command(sp)([]string{"lint-r", "--strict", "--table", "x", "--table", "y", root}, s); code != 1 || !reflect.DeepEqual(seen, []string{"x", "y"}) {
		t.Errorf("command: exit %d seen %q, want 1 (WARN under --strict) and [x y]", code, seen)
	}
}

type valuesFake struct {
	fakeRule
	seen *[]string
}

func (r valuesFake) Configure(a Args) error { *r.seen = a.Values["--table"]; return nil }

// xmlFake is a Rule whose Check fails with a positioned XML error, as an internal/xmlsrc consumer does.
type xmlFake struct{ fakeRule }

func (xmlFake) Check(f *javasrc.File) ([]Finding, error) {
	return nil, &xmlsrc.Error{Path: f.Path, Line: 3, Col: 1, Msg: "element <a> opened at 2:1 is never closed"}
}

// TestXMLErrorOnce: an *xmlsrc.Error from a rule names its file itself, so the runner prints it once
// ("<lint>: <path>:<line>:<col>: <msg>"), not behind "cannot scan <path>: <rule>:"; exit 3.
func TestXMLErrorOnce(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "x\n"})
	s, o, e := stdio()
	if code := Command(javaSpec(xmlFake{fakeRule{"lint-x", "MARK", Fail}}, One))([]string{"lint-x", root}, s); code != 3 || o.Len() != 0 ||
		e.String() != "lint-x: "+root+"/A.java:3:1: element <a> opened at 2:1 is never closed\n" {
		t.Errorf("exit %d out %q err %q", code, o, e)
	}
}

// TestCommandOptionalInAll: under --all a Spec.OptionalInAll rule that selects no file of a root is clean for
// it (the other rules still report); run by name or in a rule list it keeps the empty-root ERROR row.
func TestCommandOptionalInAll(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	js := Spec{Rule: fakeRule{"lint-js", "MARK", Warn}, Mode: Each, Usage: "u", Select: Selection{Names: []string{"*.js"}},
		OptionalInAll: true}
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each), js)
	java := "WARN  lint-a  " + root + "/A.java:1  saw: MARK\n"
	empty := "ERROR  lint-js  " + root + "  no *.js sources found (wrong path or empty source tree)\n"
	for _, tc := range []struct {
		args []string
		code int
		out  string
	}{
		{[]string{"--all", root}, 0, java},
		{[]string{"lint-js", root}, 3, empty},
		{[]string{"lint-a,lint-js", root}, 3, java + empty},
	} {
		s, o, e := stdio()
		if code := main(tc.args, s); code != tc.code || o.String() != tc.out || e.Len() != 0 {
			t.Errorf("%q: exit %d out %q err %q, want %d and %q", tc.args, code, o, e, tc.code, tc.out)
		}
	}
}

func TestParseFileFirst(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "m.md")
	if err := os.WriteFile(file, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	const use = "usage: lint-f.sh [--strict] <manual> <dir>..."
	for _, tc := range []struct {
		name   string
		args   []string
		strict bool
		roots  []string
		err    string
	}{
		{"file and roots", []string{file, dir, dir}, false, []string{dir, dir}, ""},
		{"strict first", []string{"--strict", file, dir}, true, []string{dir}, ""},
		{"a second --strict is the file", []string{"--strict", "--strict", file, dir}, false, nil, "lint-f: manual not found: --strict\n"},
		{"-- is the file", []string{"--", file, dir}, false, nil, "lint-f: manual not found: --\n"},
		{"a flag after the file is a root", []string{file, "--strict"}, false, nil, "lint-f: not a directory: --strict\n"},
		{"no root", []string{file}, false, nil, use + "\n"},
		{"strict and no root", []string{"--strict", file}, false, nil, use + "\n"},
		{"none", nil, false, nil, use + "\n"},
		{"a directory is not the file", []string{dir, dir}, false, nil, "lint-f: manual not found: " + dir + "\n"},
		{"the file before the roots", []string{dir + "/nope", dir + "/nodir"}, false, nil, "lint-f: manual not found: " + dir + "/nope\n"},
		{"every root checked", []string{file, dir, dir + "/nodir", dir + "/nodir2"}, false, nil, "lint-f: not a directory: " + dir + "/nodir\n"},
	} {
		a, err := ParseOption(FileFirst, "lint-f", use, "manual", tc.args)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != tc.err || (err == nil && (a.Option != file || a.Strict != tc.strict || !reflect.DeepEqual(a.Roots, tc.roots))) {
			t.Errorf("%s: args %+v err %q, want strict %v roots %q err %q", tc.name, a, got, tc.strict, tc.roots, tc.err)
		}
	}
}

// joinedFake is a TreeRule that reports how many files it got per call and one SUMMARY row with no path.
type joinedFake struct{ calls *[]int }

func (joinedFake) ID() string         { return "lint-j" }
func (joinedFake) Severity() Severity { return Warn }
func (r joinedFake) CheckTree(files []*javasrc.File) ([]Finding, error) {
	*r.calls = append(*r.calls, len(files))
	return []Finding{{Path: files[0].Path, Line: 1, Msg: "first"}, {Severity: Summary, Msg: fmt.Sprintf("files=%d", len(files))}}, nil
}

// TestJoinedTreeRule: a Spec.Joined TreeRule gets the files of every root in one call after the last root;
// a Summary row prints SUMMARY with no path column and never counts for the exit code, --strict included.
func TestJoinedTreeRule(t *testing.T) {
	a := tree(t, map[string]string{"A.java": "x\n"})
	b := tree(t, map[string]string{"B.java": "x\n", "C.java": "x\n"})
	var calls []int
	sp := treeSpec(joinedFake{&calls})
	sp.Joined = true
	s, o, _ := stdio()
	if code := Run([]Spec{sp}, "lint-j", Args{Roots: []string{a, b}}, s); code != 0 || !reflect.DeepEqual(calls, []int{3}) ||
		o.String() != "WARN  lint-j  "+a+"/A.java:1  first\nSUMMARY  lint-j  files=3\n" {
		t.Errorf("exit %d calls %v out %q", code, calls, o)
	}
	s, _, _ = stdio()
	if code := Run([]Spec{sp}, "lint-j", Args{Strict: true, Roots: []string{a}}, s); code != 1 {
		t.Errorf("--strict with a WARN row: exit %d, want 1", code)
	}
	f := Finding{Rule: "lint-j", Severity: Summary, Msg: "files=0"}
	if got := f.Row(); got != "SUMMARY  lint-j  files=0" {
		t.Errorf("Summary row %q", got)
	}
}

// TestFileFirstRunsOnlyAlone: a FileFirst rule is left out of --all and refused in a rule list.
func TestFileFirstRunsOnlyAlone(t *testing.T) {
	root := tree(t, map[string]string{"A.java": "MARK\n"})
	var calls []int
	ff := treeSpec(joinedFake{&calls})
	ff.Mode, ff.Option, ff.Joined = FileFirst, "manual", true
	main := Command(javaSpec(fakeRule{"lint-a", "MARK", Warn}, Each), ff)
	s, o, _ := stdio()
	if code := main([]string{"--all", root}, s); code != 0 || o.String() != "WARN  lint-a  "+root+"/A.java:1  saw: MARK\n" {
		t.Errorf("--all: exit %d out %q, want the lint-a row only", code, o)
	}
	s, o, e := stdio()
	if code := main([]string{"lint-a,lint-j", root}, s); code != 3 || o.Len() != 0 ||
		e.String() != "n4lint: lint-j runs only alone (it takes a manual operand)\n"+usage+"\n" {
		t.Errorf("rule list: exit %d out %q err %q", code, o, e)
	}
	s, o, _ = stdio()
	if code := main([]string{"lint-j", root + "/A.java", root}, s); code != 0 || o.String() != "WARN  lint-j  "+root+"/A.java:1  first\nSUMMARY  lint-j  files=1\n" {
		t.Errorf("alone: exit %d out %q", code, o)
	}
}
