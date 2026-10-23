package driverscaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Each testdata/<case>.txt is one tests/lint-driver-scaffold.bats run recorded from the bash
// lint-driver-scaffold.sh it replaces: "-- root --" (the root, relative to the tree), "-- want --"
// ("exit <n>" then the exact stdout of the bash lint, with the tree's absolute path written @DIR@) and
// one "-- <path> --" section per file of the tree.
type golden struct {
	root, want string
	files      map[string]string
}

func readGolden(t *testing.T, path string) golden {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	g := golden{files: map[string]string{}}
	name := ""
	var body strings.Builder
	flush := func() {
		switch name {
		case "":
		case "root":
			g.root = strings.TrimSuffix(body.String(), "\n")
		case "want":
			g.want = body.String()
		default:
			g.files[name] = body.String()
		}
		body.Reset()
	}
	for _, ln := range strings.SplitAfter(string(data), "\n") {
		if strings.HasPrefix(ln, "-- ") && strings.HasSuffix(ln, " --\n") {
			flush()
			name = strings.TrimSuffix(strings.TrimPrefix(ln, "-- "), " --\n")
			continue
		}
		body.WriteString(ln)
	}
	flush()
	return g
}

// runLint runs the rule through the engine as the wrapper does and returns "exit <n>\n" + stdout and stderr.
func runLint(t *testing.T, args ...string) (string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return "exit " + strconv.Itoa(code) + "\n" + o.String(), e.String()
}

func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for p, text := range files {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestGolden: every recorded bats run prints the bash lint's exact stdout and exit code, and nothing on stderr.
func TestGolden(t *testing.T) {
	paths, _ := filepath.Glob("testdata/*.txt")
	if len(paths) < 11 {
		t.Fatalf("found %d golden cases, want every bats run on a tree (>= 11)", len(paths))
	}
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			g := readGolden(t, p)
			dir := writeTree(t, g.files)
			got, stderr := runLint(t, filepath.Join(dir, g.root))
			if got = strings.ReplaceAll(got, dir, "@DIR@"); got != g.want || stderr != "" {
				t.Errorf("stderr %q\ngot\n%s\nwant\n%s", stderr, got, g.want)
			}
		})
	}
}

const (
	pingOnly = "doPing() only calls pingOk(): every device reports UP (wizard stub, silent success)"
	fail     = "FAIL  lint-driver-scaffold  @DIR@/"
)

// TestIssue282: the shapes issue #282 names, with the corrected rows. Each comment gives what the bash
// lint printed for the same file (recorded from lint-driver-scaffold.sh).
func TestIssue282(t *testing.T) {
	for name, tc := range map[string]struct{ src, want string }{
		// bash: nothing, exit 0. `\([^)]*\)` ended the annotation at f(x)'s ")", so the header named g.
		"nested-parens": {
			"class BA extends BNDevice {\n  @Ann(value = f(x), y = g(z)) public void doPing() { pingOk(); }\n}\n",
			"exit 1\n" + fail + "A.java:2  " + pingOnly + "\n"},
		// bash: the same row (the leftover ")" named nothing); kept as the issue's own example.
		"nested-parens-single": {
			"class BB extends BNDevice {\n  @Ann(value = f(x)) public void doPing() { pingOk(); }\n}\n",
			"exit 1\n" + fail + "A.java:2  " + pingOnly + "\n"},
		// bash: FAIL at line 2, the annotation's line.
		"multiline-annotation": {
			"class BF extends BNDevice {\n  @Ann(\n      value = \"a\")\n  public void doPing() {\n    pingOk();\n  }\n}\n",
			"exit 1\n" + fail + "A.java:4  " + pingOnly + "\n"},
		// bash: nothing, exit 0. The annotation's leftover "(" ... f(x)) header named f.
		"multiline-annotation-call": {
			"class BC extends BNDevice {\n  @Ann(\n      value = \"a\",\n      other = f(x))\n  public void doPing() {\n    pingOk();\n  }\n}\n",
			"exit 1\n" + fail + "A.java:5  " + pingOnly + "\n"},
		// bash: only the line-9 row. `new java.lang.Runnable() {` and `new Outer . Inner() {` read as
		// methods named Runnable and Inner, so the doPing stubs inside them were never checked.
		"qualified-anonymous": {
			"class BD {\n  Runnable r = new java.lang.Runnable() {\n    public void doPing() { pingOk(); }\n  };\n" +
				"  Object o = new Outer . Inner() {\n    public void doPing() { this.pingOk(); }\n  };\n" +
				"  Object p = new Plain() {\n    public void doPing() { pingOk(); }\n  };\n}\n",
			"exit 1\n" + fail + "A.java:3  " + pingOnly + "\n" + fail + "A.java:6  " + pingOnly + "\n" +
				fail + "A.java:9  " + pingOnly + "\n"},
		// bash: FAIL at line 2 (an unclosed "(" was header text). Now the argument list swallows the rest
		// of the file, so its boundaries are not trusted: one WARN row, never a clean pass.
		"unclosed-annotation": {
			"class BE extends BNDevice {\n  @Ann(x = 1\n  public void doPing() { pingOk(); }\n}\n",
			"exit 0\nWARN  lint-driver-scaffold  @DIR@/A.java:4  method boundaries not trusted (an annotation's argument list does not close): a stub in this file may be missed\n"},
		// Braces and parentheses inside annotation arguments, on one line or several, move nothing.
		"annotation-array": {
			"class BG extends BNDevice {\n  @Ann(v = {f(\"}\"), g(1)},\n      w = @Inner(h(2)))\n  @Other\n  public void doPing() { pingOk(); }\n}\n",
			"exit 1\n" + fail + "A.java:5  " + pingOnly + "\n"},
		// A "..." left open at its line end ends there, so the stub after it is still found, but the
		// file's boundaries are not trusted: the stub's FAIL plus one WARN naming the literal.
		"unterminated-literal": {
			"class BH extends BNDevice {\n  String s = \"abc { ;\n  public void doPing() { pingOk(); }\n}\n",
			"exit 1\n" + fail + "A.java:3  " + pingOnly + "\nWARN  lint-driver-scaffold  @DIR@/A.java:4  method boundaries not trusted (an unterminated string or char literal): a stub in this file may be missed\n"},
	} {
		t.Run(name, func(t *testing.T) {
			dir := writeTree(t, map[string]string{"A.java": tc.src})
			got, stderr := runLint(t, dir)
			if got = strings.ReplaceAll(got, dir, "@DIR@"); got != tc.want || stderr != "" {
				t.Errorf("stderr %q\ngot\n%s\nwant\n%s", stderr, got, tc.want)
			}
		})
	}
}

// TestArgs: the argument contract of lint-driver-scaffold.sh (awk_lint_parse raw: no flag is parsed,
// only the first root counts, usage exits 3), recorded from the bash lint.
func TestArgs(t *testing.T) {
	dir := writeTree(t, map[string]string{"x.txt": ""})
	file := filepath.Join(dir, "x.txt")
	missing := filepath.Join(dir, "missing")
	for _, tc := range []struct {
		args     []string
		out, err string
	}{
		{nil, "exit 3\n", "usage: lint-driver-scaffold.sh <src-root>\n"},
		{[]string{"-h"}, "exit 3\n", "lint-driver-scaffold: not a directory: -h\n"},
		{[]string{"--strict", dir}, "exit 3\n", "lint-driver-scaffold: not a directory: --strict\n"},
		{[]string{file}, "exit 3\n", "lint-driver-scaffold: not a directory: " + file + "\n"},
		{[]string{missing}, "exit 3\n", "lint-driver-scaffold: not a directory: " + missing + "\n"},
		{[]string{dir, missing}, "exit 0\n", ""},
	} {
		if out, err := runLint(t, tc.args...); out != tc.out || err != tc.err {
			t.Errorf("args %q: got %q %q, want %q %q", tc.args, out, err, tc.out, tc.err)
		}
	}
}

// TestUnreadableDir: a sub-directory the walk cannot enter is an env error (exit 3) after the rest is
// scanned, with the bash lint's message.
func TestUnreadableDir(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root")
	}
	dir := writeTree(t, map[string]string{"locked/BX.java": "class BX {}\n"})
	locked := filepath.Join(dir, "locked")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o755)
	out, err := runLint(t, dir)
	want := "lint-driver-scaffold: cannot list every file under " + dir + ": find: '" + locked + "': Permission denied\n"
	if out != "exit 3\n" || err != want {
		t.Errorf("got %q %q, want %q %q", out, err, "exit 3\n", want)
	}
}
