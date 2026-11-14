package livesmoke

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The three cases below come from the first run of live-smoke against a real 4.15 station.

func TestTheTierOptInsPassThroughToTheServer(t *testing.T) {
	a, err := ParseArgs([]string{"--apply", "--allow-tier-b", "Lab", "--allow-tier-c", "Old"})
	if err != nil {
		t.Fatal(err)
	}
	argv := strings.Join(ServerArgv(a), " ")
	for _, want := range []string{"--allow-tier-b Lab", "--allow-tier-c Old"} {
		if !strings.Contains(argv, want) {
			t.Errorf("server argv lacks %q: %s", want, argv)
		}
	}
	if !strings.Contains(helpText, "--allow-tier-b NAME") || !strings.Contains(helpText, "--allow-tier-c NAME") {
		t.Error("--help does not name the tier opt-ins")
	}
}

func TestWithoutATierOptInTheServerGetsNone(t *testing.T) {
	a, err := ParseArgs([]string{"--apply"})
	if err != nil {
		t.Fatal(err)
	}
	if argv := strings.Join(ServerArgv(a), " "); strings.Contains(argv, "--allow-tier") {
		t.Errorf("a tier opt-in the operator did not give: %s", argv)
	}
}

func TestAStepThatDidNotVerifyPrintsItsDetail(t *testing.T) {
	r := newRunner(t)
	r.fake.Do(func(st *boxstation.Station) { st.Component(st.Root(), Scratch, "baja:Folder", "") })
	_, out, _ := r.run(r.argv(true))
	if !strings.Contains(out, "failed    folder absent before\n          "+Scratch+" already exists: refusing to touch it\n") {
		t.Errorf("the console does not say why the step failed:\n%s", out)
	}
	if strings.Contains(out, "          skipped: an earlier required step") {
		t.Errorf("a skipped step repeats its detail on the console:\n%s", out)
	}
}

func TestAVerifiedStepPrintsNoDetailLine(t *testing.T) {
	r := newRunner(t)
	_, out, _ := r.run(r.argv(true))
	if strings.Contains(out, "verified  connect\n          ") {
		t.Errorf("a verified step printed a detail line:\n%s", out)
	}
}

func TestAReportInAMissingDirectoryIsExit2BeforeAnyStationContact(t *testing.T) {
	r := newRunner(t)
	missing := filepath.Join(r.tmp, "nodir")
	r.reportPath = filepath.Join(missing, "report.json")
	code, out, errOut := r.run(r.argv(true))
	want := "live-smoke: --report: directory " + missing + " does not exist"
	if code != 2 || !strings.Contains(errOut, want) || strings.Contains(errOut, ".tmp") {
		t.Errorf("exit %d\n%s%s", code, out, errOut)
	}
	if n := r.fake.Requests(); n != 0 {
		t.Errorf("%d requests reached the station before the report path was checked", n)
	}
}

func TestPlanOnlyDoesNotCheckTheReportDirectory(t *testing.T) {
	r := newRunner(t)
	r.reportPath = filepath.Join(r.tmp, "nodir", "report.json")
	if code, _, errOut := r.run(r.argv(false)); code != 0 {
		t.Errorf("plan only: exit %d\n%s", code, errOut)
	}
}
