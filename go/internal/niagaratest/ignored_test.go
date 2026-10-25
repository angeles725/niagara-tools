package niagaratest

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ignoredXML is a run with 3 PASS tests and 2 ignored ones. TestNG's XMLReporter (the one bundled in a Niagara
// test-wb.jar) adds ITestContext.getExcludedMethods() to the root ignored count, and XMLSuiteResultWriter writes
// test-method elements only for the passed, failed and skipped results: an ignored method is never named.
const ignoredXML = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
	`<testng-results skipped="0" failed="0" ignored="2" total="5" passed="3"><reporter-output/>` +
	`<suite name="S"><test name="T"><class name="test.com.example.ModuleA.BExampleStationTest">` +
	`<test-method status="PASS" name="setupStation" is-config="true"/>` +
	`<test-method status="PASS" name="a"/><test-method status="PASS" name="b"/><test-method status="PASS" name="c"/>` +
	"</class></test></suite></testng-results>\n"

const reason = "ModuleA device tests need a live controller"

// TestVerdictIgnored pins the ignored-methods gate of verdict --testng: an ignored count fails the run unless an
// allowance of exactly that count, with its reason, waives it.
func TestVerdictIgnored(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		allow []string
		exit  int
		want  []string
	}{
		{"ignored without an allowance", nil, exitFail, []string{
			"FAIL  run-niagara-test  testng: total 5, passed 3, failed 0, skipped 0, ignored 2\n",
			"FAIL  run-niagara-test  testng: 2 ignored test-methods (@Test(enabled = false) or excluded; testng-results.xml names none): " +
				"waive them with --allow-ignored-count 2 --allow-ignored-reason <why>\n",
			"FAIL  run-niagara-test  verdict: 1 FAIL reason\n"}},
		{"an allowance of exactly the ignored count", []string{"--allow-ignored-count", "2", "--allow-ignored-reason", reason}, exitPass, []string{
			"PASS  run-niagara-test  testng: total 5, passed 3, failed 0, skipped 0, ignored 2\n",
			"PASS  run-niagara-test  testng: 2 ignored test-methods allowed (--allow-ignored-count 2): " + reason + "\n",
			"PASS  run-niagara-test  verdict: 3 tests run, 0 failed, 0 skipped (testng-results)\n"}},
		{"more ignored than allowed", []string{"--allow-ignored-count", "1", "--allow-ignored-reason", reason}, exitFail, []string{
			"FAIL  run-niagara-test  testng: 2 ignored test-methods, more than the 1 allowed (" + reason + "): a new disable is never waived\n"}},
		{"fewer ignored than allowed (stale allowance)", []string{"--allow-ignored-count", "3", "--allow-ignored-reason", reason}, exitFail, []string{
			"FAIL  run-niagara-test  testng: 2 ignored test-methods, fewer than the 3 allowed: a stale allowance, lower --allow-ignored-count to 2\n"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := filepath.Join(t.TempDir(), "testng-results.xml")
			write(t, p, ignoredXML)
			var o, e bytes.Buffer
			code := run(append([]string{"verdict", "--testng", p}, c.allow...), cli.Stdio{Out: &o, Err: &e}, strings.NewReader(""))
			if code != c.exit {
				t.Errorf("exit %d, want %d:\n%s%s", code, c.exit, o.String(), e.String())
			}
			for _, w := range c.want {
				if !strings.Contains(o.String(), w) {
					t.Errorf("no row %q in:\n%s%s", w, o.String(), e.String())
				}
			}
		})
	}
}

// TestVerdictAllowanceRefusals: an allowance with no reason, or one that cannot be checked, is refused before any
// file is read (exit 2, usage).
func TestVerdictAllowanceRefusals(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"count without a reason", []string{"--testng", "x.xml", "--allow-ignored-count", "2"},
			"--allow-ignored-count needs --allow-ignored-reason <why>"},
		{"reason without a count", []string{"--testng", "x.xml", "--allow-ignored-reason", reason},
			"--allow-ignored-reason needs --allow-ignored-count <n>"},
		{"blank reason", []string{"--testng", "x.xml", "--allow-ignored-count", "2", "--allow-ignored-reason", " \t"},
			"--allow-ignored-count needs --allow-ignored-reason <why>"},
		{"reason with a line break", []string{"--testng", "x.xml", "--allow-ignored-count", "2", "--allow-ignored-reason", "a\nb"},
			"--allow-ignored-reason must be one line of printable text, 200 bytes at most"},
		{"count 0", []string{"--testng", "x.xml", "--allow-ignored-count", "0", "--allow-ignored-reason", reason},
			`--allow-ignored-count "0" is not a count from 1`},
		{"count not a number", []string{"--testng", "x.xml", "--allow-ignored-count", "2x", "--allow-ignored-reason", reason},
			`--allow-ignored-count "2x" is not a count from 1`},
		{"no value", []string{"--testng", "x.xml", "--allow-ignored-count"}, "--allow-ignored-count needs a <n>"},
		{"given twice", []string{"--testng", "x.xml", "--allow-ignored-count", "1", "--allow-ignored-count", "2"},
			"--allow-ignored-count given twice"},
		{"without --testng", []string{"--allow-ignored-count", "2", "--allow-ignored-reason", reason},
			"--allow-ignored-count needs --testng (a console log has no ignored count)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			var o, e bytes.Buffer
			code := run(append([]string{"verdict"}, c.args...), cli.Stdio{Out: &o, Err: &e}, strings.NewReader(""))
			if code != exitUsage || !strings.Contains(e.String(), "run-niagara-test: "+c.want+"\n") || o.Len() != 0 {
				t.Errorf("exit %d, want 2 with %q:\nstdout %q\nstderr %s", code, c.want, o.String(), e.String())
			}
		})
	}
}

// TestRunIgnored: run takes the same allowance, refuses one without a reason before any copy, and hands it to the
// verdict of the results test.exe wrote.
func TestRunIgnored(t *testing.T) {
	t.Parallel()
	t.Run("count without a reason is refused before any copy", func(t *testing.T) {
		t.Parallel()
		fx := newFixture(t)
		code, out := fx.run(t, append([]string{"run"}, fx.args("--allow-ignored-count", "2")...)...)
		if code != exitUsage || !strings.Contains(out, "run: --allow-ignored-count needs --allow-ignored-reason <why>") {
			t.Errorf("exit %d, want 2:\n%s", code, out)
		}
		if n := len(fx.f.of("robocopy")) + len(fx.f.of("gradle")) + len(fx.f.of("test")); n != 0 {
			t.Errorf("%d copy/build/run calls after a refusal", n)
		}
	})
	for _, c := range []struct {
		name  string
		allow []string
		exit  int
		want  string
	}{
		{"ignored without an allowance", nil, exitFail, "FAIL  run-niagara-test  testng: 2 ignored test-methods"},
		{"allowed (case (ii): no registered type, so the source disables none)", []string{"--allow-ignored-count", "2", "--allow-ignored-reason", reason}, exitPass,
			"PASS  run-niagara-test  testng: 2 ignored test-methods allowed (--allow-ignored-count 2, case (ii): the source disables 0, " +
				"TestNG ignored 2): " + reason},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			fx := newFixture(t)
			fx.f.on["test"] = func(_ context.Context, cmd Cmd) (int, error) {
				fx.f.testExe(cmd, ignoredXML, greenLog)
				fx.f.useNUH(cmd)
				return 0, nil
			}
			code, out := fx.run(t, append([]string{"run"}, fx.args(c.allow...)...)...)
			if code != c.exit || !strings.Contains(out, c.want) {
				t.Errorf("exit %d, want %d with %q:\n%s", code, c.exit, c.want, out)
			}
		})
	}
}
