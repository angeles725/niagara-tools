package niagaratest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// traceXML is a synthetic failing testng-results.xml: one PASS method and two FAIL ones, a config FAIL included,
// each FAIL with a <full-stacktrace> of lines frames ("frame <i>") that names the scratch dir, the home copy and
// the user home.
func traceXML(scratch, copyDir, home string, lines int) string {
	var trace strings.Builder
	fmt.Fprintf(&trace, "java.lang.AssertionError: expected 1 got 2\n\tat %s\\modules\\moduleA-rt.jar\n\tat %s\\out\\x\n\tat %s/.gradle/y\n",
		copyDir, scratch, home)
	for i := 1; i <= lines; i++ {
		fmt.Fprintf(&trace, "\tat test.com.example.ModuleA.BExampleStationTest.frame(BExampleStationTest.java:%d) frame %d\n", i, i)
	}
	return `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<testng-results skipped="0" failed="1" ignored="0" total="2" passed="1"><suite name="S"><test name="T">` + "\n" +
		`<class name="test.com.example.ModuleA.BExampleStationTest">` + "\n" +
		`<test-method status="FAIL" is-config="true" name="setupStation"><exception class="java.lang.IllegalStateException">` +
		`<message><![CDATA[no station]]></message><full-stacktrace><![CDATA[java.lang.IllegalStateException: no station` + "\n\tat setup frame\n" +
		`]]></full-stacktrace></exception></test-method>` + "\n" +
		`<test-method status="PASS" name="reads_value"/>` + "\n" +
		`<test-method status="FAIL" name="writes_value"><exception class="java.lang.AssertionError">` +
		`<message><![CDATA[expected 1 got 2]]></message><full-stacktrace><![CDATA[` + trace.String() +
		`]]></full-stacktrace></exception></test-method>` + "\n" +
		`</class></test></suite></testng-results>` + "\n"
}

// failWith is a test.exe that writes xml (built from the run's own paths) and fails the console log.
func failWith(fx *fixture, xml func(scratch, copyDir string) string) func(context.Context, Cmd) (int, error) {
	return func(_ context.Context, c Cmd) (int, error) {
		outDir := argValue(c.Args, "-output:")
		scratch := strings.TrimSuffix(outDir, `\out\testng`)
		fx.f.testExe(c, xml(scratch, scratch+`\home`), "Total tests run: 2, Failures: 1, Skips: 0\r\n")
		fx.f.useNUH(c)
		return 1, nil
	}
}

// kept is the run log a FAIL row names and its siblings in the log dir, by name suffix after the log's stem.
func (fx *fixture) kept(t *testing.T, out string) (log string, siblings map[string]string) {
	t.Helper()
	log = fx.keptLog(t, out)
	p := keptRe.FindStringSubmatch(out)[1]
	stem := strings.TrimSuffix(filepath.Base(p), ".log")
	es, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	siblings = map[string]string{}
	for _, e := range es {
		if e.Name() == filepath.Base(p) || !strings.HasPrefix(e.Name(), stem+".") {
			continue
		}
		siblings[strings.TrimPrefix(e.Name(), stem+".")] = readLogFile(t, filepath.Join(filepath.Dir(p), e.Name()))
	}
	return log, siblings
}

// A FAIL run keeps a redacted copy of testng-results.xml and every failed method's whole trace, a trace longer
// than the 200-line stderr tail included; the scratch dir, its home copy and the user home are redacted in both.
func TestRunFailKeepsTestNGEvidence(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.h.home = "/home/dev"
	fx.f.on["test"] = failWith(fx, func(scratch, copyDir string) string { return traceXML(scratch, copyDir, "/home/dev", 300) })
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	log, sib := fx.kept(t, out)
	results, traces := sib["testng-results.xml"], sib["traces.txt"]
	if results == "" || traces == "" || len(sib) != 3 {
		t.Fatalf("log dir siblings %q, want testng-results.xml, traces.txt and stderr.txt", keys(sib))
	}
	for _, w := range []string{`<test-method status="FAIL" name="writes_value">`, "frame 300\n", `at <copy>\modules\moduleA-rt.jar`} {
		if !strings.Contains(results, w) {
			t.Errorf("results copy lacks %q", w)
		}
	}
	for _, w := range []string{
		"== FAIL test.com.example.ModuleA.BExampleStationTest.setupStation (config) ==\n",
		"java.lang.IllegalStateException: no station\n",
		"== FAIL test.com.example.ModuleA.BExampleStationTest.writes_value ==\n",
		"java.lang.AssertionError: expected 1 got 2\n",
		"frame 1\n", "frame 150\n", "frame 300\n",
		`at <copy>\modules\moduleA-rt.jar`, `at <scratch>\out\x`, "at <you>/.gradle/y",
	} {
		if !strings.Contains(traces, w) {
			t.Errorf("traces lack %q:\n%.2000s", w, traces)
		}
	}
	if strings.Contains(traces, "reads_value") {
		t.Errorf("traces name a PASS method")
	}
	for name, body := range map[string]string{"log": log, "results": results, "traces": traces} {
		for _, nw := range []string{"run-niagara-test-", "/home/dev", `C:\Users\you`} {
			if strings.Contains(body, nw) {
				t.Errorf("%s holds %q (want it redacted)", name, nw)
			}
		}
	}
	for _, w := range []string{"testng-results.xml: copied to ", "failure traces: 2 failed methods in "} {
		if !strings.Contains(log, w) {
			t.Errorf("log lacks %q:\n%s", w, log)
		}
	}
}

// A green run keeps only its run log, even with --log-dir: no results copy, no traces, no stderr copy.
func TestRunPassKeepsNoTestNGEvidence(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	dir := filepath.Join(t.TempDir(), "logs")
	code, out := fx.run(t, append([]string{"run"}, fx.args("--log-dir", dir)...)...)
	if code != exitPass {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	es, _ := os.ReadDir(dir)
	if len(es) != 1 || !strings.HasSuffix(es[0].Name(), ".log") {
		t.Errorf("log dir holds %d entries, want the run log only", len(es))
	}
}

// A missing or unreadable testng-results.xml still leaves a run log, with a row saying it was unavailable.
func TestRunFailTestNGUnavailable(t *testing.T) {
	t.Parallel()
	for name, mk := range map[string]func(t *testing.T, outDir string){
		"missing":   func(*testing.T, string) {},
		"directory": func(t *testing.T, outDir string) { write(t, filepath.Join(outDir, "testng-results.xml", "x"), "x") },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
				mk(t, fx.f.wsl(argValue(c.Args, "-output:")))
				fx.f.useNUH(c)
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if code != exitFail {
				t.Fatalf("exit %d, want 1:\n%s", code, out)
			}
			log, sib := fx.kept(t, out)
			if !regexp.MustCompile(`(?m)^testng-results\.xml unavailable: .+$`).MatchString(log) || len(sib) != 1 || sib["stderr.txt"] != "" {
				t.Errorf("siblings %q, log:\n%s", keys(sib), log)
			}
			if strings.Contains(log, "run-niagara-test-") {
				t.Errorf("the unavailable row is not redacted:\n%s", log)
			}
		})
	}
}

// A results file over the copy cap is cut at a line end below it, with a marker row; the traces stay bounded too.
func TestRunFailTestNGEvidenceCapped(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	lines := evidenceCap/40 + 1000 // each frame line is over 40 bytes: the file is over the cap
	fx.f.on["test"] = failWith(fx, func(scratch, copyDir string) string { return traceXML(scratch, copyDir, "/nonexistent-home", lines) })
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	_, sib := fx.kept(t, out)
	results, traces := sib["testng-results.xml"], sib["traces.txt"]
	for name, body := range map[string]string{"results": results, "traces": traces} {
		if len(body) > evidenceCap+512 {
			t.Errorf("%s is %d bytes, over the %d cap", name, len(body), evidenceCap)
		}
		if !regexp.MustCompile(`\n\(truncated: [^\n]*\)\n$`).MatchString(body) {
			t.Errorf("%s lacks a final truncation marker row:\n...%s", name, body[max(len(body)-300, 0):])
		}
	}
	if !strings.Contains(traces, "frame 1\n") {
		t.Errorf("traces lack the first frames")
	}
}

// stderrRun is a test.exe that fails on traceXML and writes stderr to its stderr: the station log a failed run
// keeps whole beside its log.
func stderrRun(fx *fixture, stderr func(scratch, copyDir string) string) func(context.Context, Cmd) (int, error) {
	return func(_ context.Context, c Cmd) (int, error) {
		scratch := strings.TrimSuffix(argValue(c.Args, "-output:"), `\out\testng`)
		fx.f.testExe(c, traceXML(scratch, scratch+`\home`, "/nonexistent-home", 3), stderr(scratch, scratch+`\home`))
		fx.f.useNUH(c)
		return 1, nil
	}
}

// A FAIL run keeps test.exe's whole stderr (the station log), far past the 200-line tail of the run log, with the
// scratch dir, its home copy and the user home redacted, and names it in the kept evidence section.
func TestRunFailKeepsFullStderr(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	fx.h.home = "/home/dev"
	fx.f.on["test"] = stderrRun(fx, func(scratch, copyDir string) string {
		return numbered("station", 500) + fmt.Sprintf("Loading %s\\modules\\moduleA-rt.jar\r\nscratch %s\\out\\x\r\ngradle /home/dev/.gradle/y\r\n",
			copyDir, scratch) + "Total tests run: 2, Failures: 1, Skips: 0\r\n"
	})
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	log, sib := fx.kept(t, out)
	stderr, ok := sib["stderr.txt"]
	if !ok {
		t.Fatalf("log dir siblings %q, want a stderr.txt", keys(sib))
	}
	for _, w := range []string{"station 1\r\n", "station 2\r\n", "station 250\r\n", "station 500\r\n",
		`Loading <copy>\modules\moduleA-rt.jar`, `scratch <scratch>\out\x`, "gradle <you>/.gradle/y", "Total tests run: 2"} {
		if !strings.Contains(stderr, w) {
			t.Errorf("stderr copy lacks %q", w)
		}
	}
	for _, nw := range []string{"run-niagara-test-", "/home/dev"} {
		if strings.Contains(stderr, nw) {
			t.Errorf("stderr copy holds %q (want it redacted)", nw)
		}
	}
	if !regexp.MustCompile(`(?m)^test\.exe stderr: copied to \S+\.stderr\.txt$`).MatchString(log) {
		t.Errorf("kept evidence does not name the stderr copy:\n%s", log)
	}
}

// A stderr over the copy cap is cut at a line end below it, with a final truncation row.
func TestRunFailFullStderrCapped(t *testing.T) {
	t.Parallel()
	fx := newFixture(t)
	n := evidenceCap/12 + 1000 // "station <i>\r\n" is at least 12 bytes from i = 1000: the file is over the cap
	fx.f.on["test"] = stderrRun(fx, func(string, string) string { return numbered("station", n) })
	code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
	if code != exitFail {
		t.Fatalf("exit %d, want 1:\n%s", code, out)
	}
	_, sib := fx.kept(t, out)
	stderr := sib["stderr.txt"]
	if len(stderr) > evidenceCap+512 {
		t.Errorf("stderr copy is %d bytes, over the %d cap", len(stderr), evidenceCap)
	}
	if !regexp.MustCompile(`\r\n\(truncated: [^\n]*\)\n$`).MatchString(stderr) {
		t.Errorf("stderr copy lacks a line-end cut and a final truncation row:\n...%q", stderr[max(len(stderr)-300, 0):])
	}
	if !strings.HasPrefix(stderr, "station 1\r\nstation 2\r\n") || strings.Contains(stderr, fmt.Sprintf("station %d\r", n)) {
		t.Errorf("stderr copy does not keep the head and drop the tail")
	}
}

// A missing or unreadable stderr.txt is a "test.exe stderr unavailable" row in the log, redacted, and no copy;
// the run still ends with its log, never a crash.
func TestRunFailFullStderrUnavailable(t *testing.T) {
	t.Parallel()
	for name, mk := range map[string]func(t *testing.T, path string){
		"missing":   func(t *testing.T, path string) { rm(t, path) },
		"directory": func(t *testing.T, path string) { rm(t, path); write(t, filepath.Join(path, "x"), "x") },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["test"] = func(_ context.Context, c Cmd) (int, error) {
				fx.f.testExe(c, traceXML("s", "c", "/nonexistent-home", 3), "Total tests run: 2, Failures: 1, Skips: 0\r\n")
				mk(t, filepath.Join(filepath.Dir(fx.f.wsl(argValue(c.Args, "-output:"))), "stderr.txt"))
				fx.f.useNUH(c)
				return 1, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args()...)...)
			if code != exitEnv { // the console cross-check cannot read stderr.txt either
				t.Fatalf("exit %d, want 3:\n%s", code, out)
			}
			log, sib := fx.kept(t, out)
			if _, ok := sib["stderr.txt"]; ok || !regexp.MustCompile(`(?m)^test\.exe stderr unavailable: .+$`).MatchString(log) {
				t.Errorf("siblings %q, log:\n%s", keys(sib), log)
			}
			if strings.Contains(log, "run-niagara-test-") {
				t.Errorf("the unavailable row is not redacted:\n%s", log)
			}
		})
	}
}

func rm(t *testing.T, path string) {
	t.Helper()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func keys(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
