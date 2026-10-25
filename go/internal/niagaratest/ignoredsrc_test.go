package niagaratest

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// part writes a part directory: moduleTest-include.xml registering types (nil: no such file) and srcTest/ files
// (path under srcTest/ -> body).
func part(t *testing.T, types []string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if types != nil {
		var b strings.Builder
		b.WriteString("<types>\n")
		for _, c := range types {
			b.WriteString(`  <type class="` + c + `" name="x"/>` + "\n")
		}
		b.WriteString("</types>\n")
		write(t, filepath.Join(dir, testIncludeName), b.String())
	}
	for p, body := range files {
		write(t, filepath.Join(dir, "srcTest", p), body)
	}
	return dir
}

const pkgA = "package test.a;\nimport org.testng.annotations.Test;\n"

// TestDisabledInSource pins how the scan names the disabled test-methods of the registered types.
func TestDisabledInSource(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		types []string
		files map[string]string
		want  []string
	}{
		{"no moduleTest-include.xml registers none", nil, nil, nil},
		{"enabled tests, JUnit @Ignore and a commented-out disable are not ignored", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne extends BTestNg {\n  @Test public void a() {}\n" +
				"  @Ignore @org.junit.Test public void b() {}\n  // @Test(enabled = false)\n  public void c() {}\n" +
				"  @Test(enabled = true, description = \"enabled = false\") public void d() {}\n}\n"}, nil},
		{"method-level forms, other annotations and a generic result", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne extends BTestNg {\n" +
				"  @Test(\n    groups = {\"slow\"},\n    enabled\n      = false)\n  public void a() {}\n" +
				"  @SuppressWarnings(\"x\") @Test(enabled=false) @Deprecated\n  public final java.util.List<String> b() { return null; }\n" +
				"  @Test(enabled = false) public void c(int n) {}\n  @Test(enabled = false) public void c() {}\n}\n"},
			[]string{"BOne.a", "BOne.b", "BOne.c", "BOne.c"}},
		{"inherited through an abstract srcTest base, counted on each concrete type, a nearer declaration wins",
			[]string{"test.a.BBase", "test.a.BOne", "test.a.BTwo"}, map[string]string{
				"test/a/BBase.java": pkgA + "public abstract class BBase extends BTestNg {\n" +
					"  @Test(enabled = false) public void base() {}\n  @Test(enabled = false) public void over() {}\n}\n",
				"test/a/BOne.java": pkgA + "public class BOne extends BBase {\n  @Test public void over() {}\n}\n",
				"test/a/BTwo.java": pkgA + "public class BTwo extends test.a.BBase {}\n"},
			[]string{"BOne.base", "BTwo.base", "BTwo.over"}},
		{"a nested registered type and a method of a nested or anonymous class",
			[]string{"test.a.BOne$Inner", "test.a.BOne"}, map[string]string{
				"test/a/BOne.java": pkgA + "public class BOne extends BTestNg {\n" +
					"  public static class Inner extends BTestNg { @Test(enabled = false) public void in() {} }\n" +
					"  Object o = new Object() { @Test(enabled = false) public void anon() {} };\n}\n"},
			[]string{"BOne$Inner.in"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := disabledInSource(part(t, c.types, c.files))
			if err != nil || !slices.Equal(got, c.want) {
				t.Errorf("got %q, %v; want %q", got, err, c.want)
			}
		})
	}
}

// TestDisabledInSourceRefusals: what the scan cannot name exactly is an error naming the file, never a guess.
func TestDisabledInSourceRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		types []string
		files map[string]string
		want  string
	}{
		{"class-level disable", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "@Test(enabled = false)\npublic class BOne extends BTestNg { public void a() {} }\n"},
			"BOne.java: class-level @Test(enabled = false) on BOne disables every public method"},
		{"class-level disable on a srcTest base", []string{"test.a.BOne"}, map[string]string{
			"test/a/BBase.java": pkgA + "@Test(enabled = false) public abstract class BBase extends BTestNg {}\n",
			"test/a/BOne.java":  pkgA + "public class BOne extends BBase {}\n"},
			"BBase.java: class-level @Test(enabled = false) on BBase"},
		{"enabled from a constant", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne extends BTestNg {\n  @Test(enabled = OFF) public void a() {}\n}\n"},
			"BOne.java:4: BOne.a: @Test enabled is not a true or false literal"},
		{"unterminated text block", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne { String s = \"\"\"\n never closed\n"},
			`BOne.java ends inside a """ text block`},
		{"registered type with no source", []string{"test.a.BGone"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne {}\n"},
			"cannot name the disabled test-methods of test.a.BGone: no declaration under "},
		{"superclass hidden in an unreadable file", []string{"test.a.BOne"}, map[string]string{
			"test/a/BOne.java":  pkgA + "public class BOne extends BBase {}\n",
			"test/a/BBase.java": "/* never closed\n"},
			"superclass BBase of BOne is not found, and the scanner cannot read "},
		{"two registered types of one simple name", []string{"test.a.BOne", "test.b.BOne"}, map[string]string{
			"test/a/BOne.java": pkgA + "public class BOne {}\n", "test/b/BOne.java": "package test.b;\npublic class BOne {}\n"},
			"registered types test.a.BOne and test.b.BOne share the simple name BOne"},
		{"malformed moduleTest-include.xml", nil, nil, testIncludeName},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := part(t, c.types, c.files)
			if c.types == nil {
				write(t, filepath.Join(dir, testIncludeName), "<types><type class=\"x\"></types>\n")
			}
			got, err := disabledInSource(dir)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %q, %v; want an error with %q", got, err, c.want)
			}
		})
	}
}

// TestRunAllowanceByNameRefusals: --allow-ignored values are checked at parse (exit 2).
func TestRunAllowanceByNameRefusals(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no reason", []string{"--allow-ignored", "BOne.a"}, "--allow-ignored needs --allow-ignored-reason <why>"},
		{"not Class.method", []string{"--allow-ignored", "a", "--allow-ignored-reason", reason}, `"a" is not <Class>.<method>`},
		{"given twice", []string{"--allow-ignored", "BOne.a", "--allow-ignored", "BOne.a", "--allow-ignored-reason", reason},
			"BOne.a given twice"},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			code, out := fx.run(t, append([]string{"run"}, fx.args(c.args...)...)...)
			if code != exitUsage || !strings.Contains(out, c.want) || len(fx.f.calls) != 0 {
				t.Errorf("exit %d, %d calls, want 2 with %q:\n%s", code, len(fx.f.calls), c.want, out)
			}
		})
	}
}
