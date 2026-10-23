package extwritable

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/ext-writable-shape.bats run on a tree and every edge tree, recorded from the bash
// lint (the empty-root and unreadable cases are table tests below).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 40) }

// run runs the rule alone, as its wrapper does, and returns the exit code, stdout and stderr with dir
// spelled @DIR@.
func run(t *testing.T, dir string, args ...string) (int, string, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := lint.Command(Spec())(append([]string{ID}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, strings.ReplaceAll(o.String(), dir, "@DIR@"), strings.ReplaceAll(e.String(), dir, "@DIR@")
}

// write creates the files of a tree under a new temporary directory.
func write(t *testing.T, files map[string]string) string {
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

// room is a class whose line 2 declares an OPERATOR BStatusNumeric no action writes.
const room = "package a;\n@NiagaraProperty(name=\"sp\", type=\"BStatusNumeric\", flags=Flags.OPERATOR)\npublic class BA {}\n"

// TestEmptyRoot is EW11's empty directory (a golden case cannot hold one): the K20 ERROR row, exit 3.
func TestEmptyRoot(t *testing.T) {
	dir := t.TempDir()
	code, out, stderr := run(t, dir, dir)
	want := "ERROR  ext-writable-shape  @DIR@  no Java sources found (K20: wrong path or empty scaffold)\n"
	if code != 3 || out != want || stderr != "" {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, out, stderr, want)
	}
}

// TestLocked is EW-locked: an unreadable sub-directory stops the scan with exit 3 and no rows, as
// scan_files_or_exit did, even beside a file that WARNs.
func TestLocked(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: chmod 000 does not block reads")
	}
	dir := write(t, map[string]string{"A.java": room})
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	code, out, stderr := run(t, dir, "--strict", dir)
	want := ID + ": cannot list every file under @DIR@: find: '@DIR@/locked': Permission denied\n"
	if code != 3 || out != "" || stderr != want {
		t.Errorf("exit %d out %q err %q, want 3, no rows and %q", code, out, stderr, want)
	}
}

// prop is an OPERATOR BStatusNumeric @NiagaraProperty named name, with extra attributes before flags.
func prop(name, extra string) string {
	return `@NiagaraProperty(name="` + name + `", type="BStatusNumeric", ` + extra + `flags=Flags.OPERATOR)` + "\n"
}

// TestLiteralComments: a "//" or "/*" inside a string literal is not a comment [ev: retro
// mb-strip-string-literals Δ1]. The awk's private strip cut the line there, so a property's flags were lost,
// a "/*" hid every property up to the next "*/", and an action's name was never harvested.
func TestLiteralComments(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"a // in a property string keeps its flags",
			prop("a", `defaultValue="http://x", `), "a@2"},
		{"a /* in a property string hides no later property",
			prop("b", `defaultValue="/*", `) + prop("c", "") + prop("d", `defaultValue="*/", `), "b@2 c@3 d@4"},
		{"a // in an action string keeps the next action's name",
			prop("e", "") + `@NiagaraAction(name="go", defaultValue="BString.make(\"//\")")` + "\n" +
				`@NiagaraAction(name="setE")` + "\n", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := write(t, map[string]string{"B.java": "package a;\n" + c.src + "public class B {}\n"})
			if got := warned(t, dir); got != c.want {
				t.Errorf("WARN rows %q, want %q", got, c.want)
			}
		})
	}
}

// warned runs the rule on dir and returns its WARN rows as "<slot>@<line>", blank-separated.
func warned(t *testing.T, dir string) string {
	t.Helper()
	code, out, stderr := run(t, dir, dir)
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d err %q", code, stderr)
	}
	var got []string
	for _, row := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
		if row == "" {
			continue
		}
		f := strings.Fields(row) // WARN ext-writable-shape <path>:<line> <slot>: ...
		got = append(got, strings.TrimSuffix(f[3], ":")+"@"+f[2][strings.LastIndexByte(f[2], ':')+1:])
	}
	return strings.Join(got, " ")
}

// TestLiteralCode: the text inside a string literal is never code. The awk counted a parenthesis in a
// string, so an unbalanced one joined the next annotations into one (their rows were lost); it read an
// "@NiagaraProperty(" in a string as an annotation that swallowed the real one after it; and it read a
// setX( in a do<Action> body's string as a write that exempted x.
func TestLiteralCode(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"a ( in a property string joins no later property",
			prop("d", `defaultValue="f(", `) + prop("e", "") + prop("f", `defaultValue="g())", `), "d@2 e@3 f@4"},
		{"a ( in an action string hides no later action",
			prop("q", "") + `@NiagaraAction(name="go", defaultValue="f(")` + "\n" + `@NiagaraAction(name="setQ")` + "\n", ""},
		{"an @NiagaraProperty( in a string is not an annotation",
			"class D { String s = \"@NiagaraProperty(\"; }\n" + prop("x", ""), "x@3"},
		{"a setX( in a do-body string is not a write",
			prop("x", "") + `@NiagaraAction(name="go")` + "\n" +
				"public class C {\n  public void doGo()\n  {\n    log(\"setX(\");\n  }\n}\n", "x@2"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := write(t, map[string]string{"B.java": "package a;\n" + c.src + "public class B {}\n"})
			if got := warned(t, dir); got != c.want {
				t.Errorf("WARN rows %q, want %q", got, c.want)
			}
		})
	}
}
