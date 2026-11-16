package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// TestGateAndReport: the verify gate's and report-module's exit statuses map to the chain's exit code.
func TestGateAndReport(t *testing.T) {
	cases := []struct {
		name                 string
		gateExit, reportExit string
		flags                []string
		code                 int
		stderr               string
	}{
		{"both pass", "0", "0", nil, ExitOK, ""},
		{"the gate fails", "1", "0", nil, ExitGate, "build.sh: verify gate failed — do not deploy these jars\n"},
		{"the gate fails with --no-report", "1", "0", []string{"--no-report"}, ExitGate, "build.sh: verify gate failed — do not deploy these jars\n"},
		{"--no-report skips a failing report", "0", "1", []string{"--no-report"}, ExitOK, ""},
		{"the report has FAILs", "0", "1", nil, ExitGate, "build.sh: report-module punch-list has FAILs — not hand-off-ready (--no-report to skip)\n"},
		{"the report has an environment error", "0", "3", nil, ExitEnv, "build.sh: report-module environment error (exit 3)\n"},
		{"the report is INCOMPLETE", "0", "4", nil, ExitReportPartial, "build.sh: report-module verdict INCOMPLETE — a floor check"},
		{"any other report status is a FAIL", "0", "77", nil, ExitGate, "build.sh: report-module punch-list has FAILs"},
	}
	for _, c := range cases {
		f := newFixture(t)
		f.does(gateScript, "echo gate-out; echo gate-err >&2; exit "+c.gateExit)
		f.does(reportScript, "exit "+c.reportExit)
		f.wants(c.name, f.build(c.flags...), c.code, c.stderr)
		if !strings.Contains(f.out, "==> verify gate (verify-module.sh):\ngate-out\n") || !strings.HasPrefix(f.err, "gate-err\n") {
			t.Errorf("%s: the gate's streams were not passed through: %q %q", c.name, f.out, f.err)
		}
		reported := c.gateExit == "0" && len(c.flags) == 0
		if got := f.call(reportScript) != ""; got != reported {
			t.Errorf("%s: report-module ran = %v, want %v", c.name, got, reported)
		}
		if got := strings.Contains(f.out, "==> report-module (hand-off punch-list)\n"); got != reported {
			t.Errorf("%s: report row = %v, want %v", c.name, got, reported)
		}
	}
}

// TestGateArgumentsWithoutFlags: the gate gets the sources and every built jar, report-module the sources and
// the niagara_home of the build as --niagara-home, so its template lint knows the stock modules (retro K8).
func TestGateArgumentsWithoutFlags(t *testing.T) {
	f := newFixture(t)
	f.wants("default", f.build(), ExitOK)
	if want := "verify-module.sh <--src> <" + f.root + "/Foo> <" + f.built("rt") + "> <" + f.built("ux") + ">"; f.call(gateScript) != want {
		t.Errorf("gate call %q, want %q", f.call(gateScript), want)
	}
	if want := "report-module.sh <" + f.root + "/Foo> <--niagara-home> <" + f.home + ">"; f.call(reportScript) != want {
		t.Errorf("report call %q, want %q", f.call(reportScript), want)
	}
}

// TestAMissingToolIsReported: a script the chain cannot start ends as bash's status for it does.
func TestAMissingToolIsReported(t *testing.T) {
	for script, code := range map[string]int{preflightScript: ExitEnv, reportScript: ExitGate} {
		f := newFixture(t)
		path := f.kit + "/" + script
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		f.wants(script, f.build(), code, "build.sh: "+path+": No such file or directory\n")
	}
}
