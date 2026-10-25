package niagaratest

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// The registered test type of these fixtures: its source under the part's srcTest/ and its moduleTest-include.xml.
const (
	alphaType = "test.com.example.moduleA.BAlphaTest"
	alphaRel  = "ModuleA-rt/srcTest/test/com/example/moduleA/BAlphaTest.java"
)

// alphaSource is BAlphaTest with test-methods a, b and c; each named in disabled carries a @Test(enabled = false)
// in one of the forms a source writes (one line, with a description, across lines).
func alphaSource(disabled ...string) string {
	forms := map[string]string{
		"a": "@Test(enabled = false)",
		"b": "@Test(enabled=false, description = \"needs a live controller\")",
		"c": "@org.testng.annotations.Test(\n      description = \"enabled = false is only text here\",\n      enabled = false)",
	}
	var b strings.Builder
	b.WriteString("package test.com.example.moduleA;\n\nimport org.testng.annotations.Test;\n\n@NiagaraType\n" +
		"public class BAlphaTest extends BTestNg\n{\n  @Test\n  public void runs() {}\n")
	for _, m := range []string{"a", "b", "c"} {
		ann := "@Test"
		for _, d := range disabled {
			if d == m {
				ann = forms[m]
			}
		}
		fmt.Fprintf(&b, "  %s\n  public void %s() {}\n", ann, m)
	}
	b.WriteString("}\n")
	return b.String()
}

// classDisabledSource is BAlphaTest disabled at class level, which the scan refuses.
const classDisabledSource = "package test.com.example.moduleA;\n\nimport org.testng.annotations.Test;\n\n" +
	"@Test(enabled = false)\npublic class BAlphaTest extends BTestNg\n{\n  public void a() {}\n}\n"

// registerAlpha writes the part's moduleTest-include.xml registering BAlphaTest, and its source.
func (fx *fixture) registerAlpha(t *testing.T, source string) {
	t.Helper()
	write(t, filepath.Join(fx.gradle, "ModuleA-rt", "moduleTest-include.xml"),
		"<types>\n  <type class=\""+alphaType+"\" name=\"AlphaTest\"/>\n</types>\n")
	write(t, filepath.Join(fx.gradle, alphaRel), source)
}

// ignoredResults is a green testng-results.xml with passed PASS methods and a root ignored count of ignored.
func ignoredResults(passed, ignored int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `<testng-results skipped="0" failed="0" ignored="%d" total="%d" passed="%d"><suite name="S"><test name="T">`+
		`<class name="%s">`, ignored, passed+ignored, passed, alphaType)
	for i := range passed {
		fmt.Fprintf(&b, `<test-method status="PASS" name="p%d"/>`, i)
	}
	b.WriteString("</class></test></suite></testng-results>\n")
	return b.String()
}

// TestRunIgnoredNamedFromSource pins the source half of the ignored gate: run names the @Test(enabled = false)
// methods of the registered types from the source under --gradle-root, FAILs when their count is not TestNG's root
// ignored, and waives them only by name (--allow-ignored <Class.method>), so a swap of one disable for another is
// caught although the count stays the same. The count pair is the fallback for a source that cannot name the set:
// the scan still runs and its names are printed, and the count is accepted only when the scan refused (case (i)) or
// its count is not TestNG's (case (ii)); a scan that names every ignored method FAILs the count (use the names).
func TestRunIgnoredNamedFromSource(t *testing.T) {
	t.Parallel()
	const why = "--allow-ignored-reason"
	cases := []struct {
		name     string
		source   string
		ignored  int
		args     []string
		exit     int
		want     []string
		noBuild  bool // refused or failed before gradle runs
		notWant  []string
		buildRun bool
	}{
		{"two disabled in source, no allowance", alphaSource("a", "b"), 2, nil, exitFail, []string{
			"PASS  run-niagara-test  source: the registered types disable 2 test-methods: BAlphaTest.a, BAlphaTest.b",
			"FAIL  run-niagara-test  testng: 2 new disabled test-methods not allowed: BAlphaTest.a, BAlphaTest.b"}, false, nil, true},
		{"allowed by name, exactly the disabled set", alphaSource("a", "c"), 2,
			[]string{"--allow-ignored", "BAlphaTest.c", "--allow-ignored", "BAlphaTest.a", why, reason}, exitPass, []string{
				"PASS  run-niagara-test  source: the registered types disable 2 test-methods: BAlphaTest.a, BAlphaTest.c",
				"PASS  run-niagara-test  testng: 2 ignored test-methods allowed by name (--allow-ignored): BAlphaTest.a, BAlphaTest.c: " + reason},
			false, []string{"FAIL  run-niagara-test  testng"}, true},
		{"the swap: allowance names a, source now disables b, count still 1", alphaSource("b"), 1,
			[]string{"--allow-ignored", "BAlphaTest.a", why, reason}, exitFail, []string{
				"FAIL  run-niagara-test  testng: 1 new disabled test-method not allowed: BAlphaTest.b",
				"FAIL  run-niagara-test  testng: 1 stale --allow-ignored name: BAlphaTest.a"}, false, nil, true},
		{"source disables 1, TestNG ignored 3", alphaSource("a"), 3,
			[]string{"--allow-ignored", "BAlphaTest.a", why, reason}, exitFail, []string{
				"FAIL  run-niagara-test  testng: 3 ignored test-methods, but the registered types' source disables 1 (BAlphaTest.a): " +
					"source and TestNG disagree"}, false, nil, true},
		// The count fallback runs the scan too, and is accepted only when the source cannot name the set.
		{"(a) the swap under the count fallback: source now disables b, count 1, TestNG 1", alphaSource("b"), 1,
			[]string{"--allow-ignored-count", "1", why, reason}, exitFail, []string{
				"PASS  run-niagara-test  source: the registered types disable 1 test-method: BAlphaTest.b",
				"FAIL  run-niagara-test  testng: --allow-ignored-count 1 refused: the source names every ignored test-method " +
					"(BAlphaTest.b): use --allow-ignored names"}, false, []string{"PASS  run-niagara-test  testng: 1 ignored"}, true},
		{"(b) the count fallback with the exact count, the source names them", alphaSource("a", "b"), 2,
			[]string{"--allow-ignored-count", "2", why, reason}, exitFail, []string{
				"PASS  run-niagara-test  source: the registered types disable 2 test-methods: BAlphaTest.a, BAlphaTest.b",
				"FAIL  run-niagara-test  testng: --allow-ignored-count 2 refused: the source names every ignored test-method " +
					"(BAlphaTest.a, BAlphaTest.b): use --allow-ignored names"}, false, []string{"PASS  run-niagara-test  testng: 2 ignored"}, true},
		{"(c) case (i): a class-level disable refuses the scan, the count fallback applies", classDisabledSource, 1,
			[]string{"--allow-ignored-count", "1", why, reason}, exitPass, []string{
				"NOTE  run-niagara-test  source: cannot name the disabled test-methods of " + alphaType + ": ",
				"class-level @Test(enabled = false) on BAlphaTest",
				"; the count fallback applies (case (i): the source scan refused)",
				"PASS  run-niagara-test  testng: 1 ignored test-method allowed (--allow-ignored-count 1, case (i): the source scan refused): " +
					reason}, false, []string{"FAIL  run-niagara-test"}, true},
		{"(c) case (i): an unparseable source under the count fallback", "/* never closed\n", 1,
			[]string{"--allow-ignored-count", "1", why, reason}, exitPass, []string{
				"NOTE  run-niagara-test  source: cannot name the disabled test-methods of " + alphaType + ": ",
				"PASS  run-niagara-test  testng: 1 ignored test-method allowed (--allow-ignored-count 1, case (i): the source scan refused): " +
					reason}, false, nil, true},
		{"(d) case (ii): source disables 1, TestNG ignored 3, count 3", alphaSource("a"), 3,
			[]string{"--allow-ignored-count", "3", why, reason}, exitPass, []string{
				"PASS  run-niagara-test  source: the registered types disable 1 test-method: BAlphaTest.a",
				"PASS  run-niagara-test  testng: 3 ignored test-methods allowed (--allow-ignored-count 3, case (ii): the source disables 1, " +
					"TestNG ignored 3): " + reason}, false, []string{"FAIL  run-niagara-test"}, true},
		{"(d) case (ii) still judges the count: TestNG ignored 3, count 2", alphaSource("a"), 3,
			[]string{"--allow-ignored-count", "2", why, reason}, exitFail, []string{
				"FAIL  run-niagara-test  testng: 3 ignored test-methods, more than the 2 allowed"}, false, nil, true},
		{"(f) no allowance and no disable: the report names none", alphaSource(), 0, nil, exitPass, []string{
			"PASS  run-niagara-test  source: the registered types disable 0 test-methods"}, false, nil, true},
		{"names and a count mixed", alphaSource("a"), 1,
			[]string{"--allow-ignored", "BAlphaTest.a", "--allow-ignored-count", "1", why, reason}, exitUsage, []string{
				"run: --allow-ignored names the methods and --allow-ignored-count counts them: give one, not both"}, true, nil, false},
		{"an unparseable registered source", "package test.com.example.moduleA;\n/* never closed\npublic class BAlphaTest {}\n", 0, nil,
			exitEnv, []string{"FAIL  run-niagara-test  source: cannot name the disabled test-methods of " + alphaType + ": "}, true, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.registerAlpha(t, c.source)
			fx.f.on["test"] = func(_ context.Context, cmd Cmd) (int, error) {
				fx.f.testExe(cmd, ignoredResults(3, c.ignored), greenLog)
				fx.f.useNUH(cmd)
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args(c.args...)...)...)
			if code != c.exit {
				t.Errorf("exit %d, want %d:\n%s", code, c.exit, out)
			}
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("no row %q in:\n%s", w, out)
				}
			}
			for _, w := range c.notWant {
				if strings.Contains(out, w) {
					t.Errorf("unexpected %q in:\n%s", w, out)
				}
			}
			if c.exit == exitEnv && !strings.Contains(out, filepath.Join(fx.gradle, alphaRel)) {
				t.Errorf("the env FAIL does not name the source file:\n%s", out)
			}
			if n := len(fx.f.of("gradle")); c.noBuild && n != 0 {
				t.Errorf("%d gradle calls, want none", n)
			} else if c.buildRun && n == 0 {
				t.Errorf("no gradle call:\n%s", out)
			}
		})
	}
}
