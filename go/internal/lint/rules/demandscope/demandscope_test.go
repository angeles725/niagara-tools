package demandscope

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/demand-in-scope.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 37) }

// TestLocked is DIS-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_or_exit did, even beside a file that WARNs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class A {\n  int step(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, "--strict", dir}, cli.Stdio{Out: &o, Err: &e})
	want := ID + ": cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n"
	if code != 3 || o.String() != "" || e.String() != want {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, o.String(), e.String(), want)
	}
}

// rows runs the rule alone on one file holding src and returns its stdout with the path cut to "A.java".
func rows(t *testing.T, src string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "A.java"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	return strings.ReplaceAll(o.String(), dir+"/", "")
}

// TestCommentsAndLiterals: comment and literal text is never code. The awk cut each line at its first
// "//" (inside a string too) and read block comments, strings and chars as code, so a demand word in a
// comment cleared the whole file, a brace in a literal shifted the depth and a string held a fake read.
func TestCommentsAndLiterals(t *testing.T) {
	warn := func(line int, m, pv string) string {
		return "WARN  demand-in-scope  A.java:" + strconv.Itoa(line) + "  " + m + " reads " + pv + " with no demand-shaped input in scope\n"
	}
	for _, tc := range []struct{ name, src, want string }{
		{"block comment field", "class A {\n  /* int demandCount; */\n  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n",
			warn(3, "s", "suction")},
		{"javadoc field", "class A {\n  /**\n   * enable; */\n  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n",
			warn(4, "s", "suction")},
		{"string field", "class A {\n  String n = \"callCount\";\n  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n",
			warn(3, "s", "suction")},
		{"url keeps the rest", "class A {\n  String u = \"http://x\"; int demand;\n  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n", ""},
		{"brace in string", "class A {\n  int s(double suction) { String b = \"}\"; int target = 0; if (suction > 1) target = 1; return target; }\n  int t(double temp) { int target = 0; if (temp > 1) target = 1; return target; }\n}\n",
			warn(2, "s", "suction") + warn(3, "t", "temp")},
		{"brace in char", "class A {\n  int s(double head) { char c = '{'; int target = 0; if (head > 1) target = 1; return target; }\n}\n",
			warn(2, "s", "head")},
		{"read in string", "class A {\n  void s(double t) { setX(\"temp > 2\"); }\n}\n", ""},
		{"write in comment", "class A {\n  void s(double t) { if (temp > 2) log(); // target = 1\n  }\n}\n", ""},
		{"escaped quote", "class A {\n  void s(double t) { String q = \"\\\"// \"; if (temp > 2) setX(1); }\n}\n",
			warn(2, "s", "temp")},
	} {
		if got := rows(t, tc.src); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

// TestKeywords: a class, interface or enum declaration at class scope is neither a field nor a method.
// The awk wrote those checks as /\bclass\b/, which gawk reads as backspaces, so a nested class named
// Demand counted as a demand field and a method opened on an enum's line was checked at class scope.
func TestKeywords(t *testing.T) {
	const step = "  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n"
	for _, tc := range []struct{ name, src, want string }{
		{"nested class", "class A {\n  class Demand { int x; }\n" + step + "}\n",
			"WARN  demand-in-scope  A.java:3  s reads suction with no demand-shaped input in scope\n"},
		{"nested interface", "class A {\n  interface Enable { int X = 1; }\n" + step + "}\n",
			"WARN  demand-in-scope  A.java:3  s reads suction with no demand-shaped input in scope\n"},
		{"enum line", "class A {\n  int demand;\n  enum E { A; int f(double temp) { cmd[0] = temp > 2 ? 1 : 0; return 0; } }\n}\n", ""},
		{"enum line, no demand", "class A {\n  enum E { A; int f(double temp) { cmd[0] = temp > 2 ? 1 : 0; return 0; } }\n}\n", ""},
		{"word inside a name", "class A {\n  int subclassCount;\n" + step + "}\n", ""},
	} {
		if got := rows(t, tc.src); got != tc.want {
			t.Errorf("%s: got\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

// TestUnreadableFile pins the wrapper header: an unreadable .java file is exit 3 with a "cannot scan"
// line, and the other files are still scanned (the bash lint printed awk's error and exited 0).
func TestUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := t.TempDir()
	src := "class A {\n  int s(double suction) { int target = 0; if (suction > 1) target = 1; return target; }\n}\n"
	a, u := filepath.Join(dir, "A.java"), filepath.Join(dir, "U.java")
	if err := os.WriteFile(a, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(u, []byte(src), 0); err != nil {
		t.Fatal(err)
	}
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	row := "WARN  demand-in-scope  " + a + ":2  s reads suction with no demand-shaped input in scope\n"
	errLine := ID + ": cannot scan " + u + ": awk: fatal: cannot open file `" + u + "' for reading: Permission denied\n"
	if code != 3 || o.String() != row || e.String() != errLine {
		t.Errorf("exit %d out %q err %q, want 3, %q and %q", code, o.String(), e.String(), row, errLine)
	}
}
