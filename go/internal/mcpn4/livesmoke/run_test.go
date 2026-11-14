package livesmoke

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestDefaultPrintsTheScenarioAndNeverTouchesTheStation(t *testing.T) {
	r := newRunner(t)
	code, out, errOut := r.run(r.argv(false))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if r.fake.Requests() != 0 || r.fake.Has(Scratch) {
		t.Errorf("plan-only reached the station: %d requests", r.fake.Requests())
	}
	for _, needle := range []string{"McpSmoke", "Temp", "GreaterThan", "n4_rollback", "plan only"} {
		if !strings.Contains(out, needle) {
			t.Errorf("plan misses %q", needle)
		}
	}
	if _, err := os.Stat(filepath.Join(r.tmp, "state")); !os.IsNotExist(err) {
		t.Errorf("plan-only created the state dir: %v", err)
	}
}

func TestPlanOnlyNeedsNoStationArguments(t *testing.T) {
	code, out, _ := newRunner(t).run(nil)
	if code != 0 || !strings.Contains(out, "plan only") {
		t.Errorf("exit %d\n%s", code, out)
	}
}

func TestApplyRequiresStationAndStationHome(t *testing.T) {
	r := newRunner(t)
	for _, argv := range [][]string{{"--apply"}, {"--apply", "--station", "A=http://x"},
		{"--apply", "--station-home", "A=/x"},
		{"--apply", "--station", "A=http://x", "--station-home", "B=/x"}} {
		code, _, errOut := r.run(argv)
		if code != 2 || !strings.Contains(errOut, "--station") {
			t.Errorf("%v: exit %d, %q", argv, code, errOut)
		}
	}
}

func TestAFullRunIsVerifiedExit0AndLeavesNoFolder(t *testing.T) {
	r := newRunner(t)
	code, out, errOut := r.run(r.argv(true))
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, out, errOut)
	}
	rep := r.report()
	if rep["ok"] != true || rep["station"] != "SmokeStation" {
		t.Errorf("ok=%v station=%v", rep["ok"], rep["station"])
	}
	required := 0
	for _, s := range steps(rep) {
		if s["required"] == true {
			required++
			if s["verdict"] != "verified" {
				t.Errorf("required step %v: %v", s["name"], s["verdict"])
			}
		}
	}
	if required == 0 {
		t.Error("no required step")
	}
	if r.fake.Has(Scratch) || r.fake.Saves() < 2 {
		t.Errorf("folder present=%v saves=%d", r.fake.Has(Scratch), r.fake.Saves())
	}
}

func TestEveryWriteStepDoesDryRunThenTokenThenExecute(t *testing.T) {
	r := newRunner(t)
	r.run(r.argv(true))
	writes := 0
	for _, s := range steps(r.report()) {
		tool, _ := s["tool"].(string)
		if !strings.HasPrefix(tool, "n4_") || ReadTools[tool] || s["verdict"] == "skipped" {
			continue
		}
		writes++
		if s["dry_run_done"] != true || s["batch_id"] == nil {
			t.Errorf("write step %v: dry_run_done=%v batch_id=%v", s["name"], s["dry_run_done"], s["batch_id"])
		}
	}
	if writes < 12 {
		t.Errorf("%d write steps, want >= 12", writes)
	}
}

// The rollback of the removed folder (a nested subtree with its links) was probe-only until it
// verified on a live 4.15.6.20 station; it now gates the exit code like every other step.
func TestTheRollbackStepIsRequiredAndNothingIsProbeOnly(t *testing.T) {
	r := newRunner(t)
	_, out, _ := r.run(r.argv(true))
	rep := r.report()
	var rollback map[string]any
	for _, s := range steps(rep) {
		if s["probe"] == true {
			t.Errorf("probe step left: %v", s["name"])
		}
		if s["tool"] == "n4_rollback" {
			rollback = s
		}
	}
	if rollback == nil || rollback["required"] != true || rollback["verdict"] != "verified" {
		t.Fatalf("rollback step: %v", rollback)
	}
	if list, ok := rep["probe_only"].([]any); !ok || len(list) != 0 {
		t.Errorf("probe_only = %v", rep["probe_only"])
	}
	if strings.Contains(out, "probe-only") || strings.Contains(PlanText(), "probe-only") {
		t.Error("the plan or the run still announces a probe-only step")
	}
}

func TestTheScenarioCoversTheB1199Thermostat(t *testing.T) {
	r := newRunner(t)
	r.run(r.argv(true))
	var got []string
	for _, s := range steps(r.report()) {
		got = append(got, s["name"].(string))
	}
	for _, want := range []string{"connect", "create Folder McpSmoke", "create Compare", "link Compare.out->Cooling.in10",
		"find_dangling_outputs == [Cooling]", "read outputs == true", "read outputs == false",
		"remove folder (again)", "folder absent"} {
		if !contains(got, want) {
			t.Errorf("no step %q in %v", want, got)
		}
	}
}

func TestAPreExistingFolderAbortsBeforeAnyWrite(t *testing.T) {
	r := newRunner(t)
	r.fake.Do(func(st *boxstation.Station) { st.Component(st.Root(), Scratch, "baja:Folder", "") })
	code, _, _ := r.run(r.argv(true))
	if code != 1 || !r.fake.Has(Scratch) || r.fake.Saves() != 0 || r.report()["ok"] != false {
		t.Errorf("exit %d present=%v saves=%d ok=%v", code, r.fake.Has(Scratch), r.fake.Saves(), r.report()["ok"])
	}
}

func TestAStepMismatchExitsNonzeroSkipsTheRestAndCleansUp(t *testing.T) {
	r := newRunner(t)
	r.fake.Do(func(*boxstation.Station) { r.fake.compute = false })
	code, _, _ := r.run(r.argv(true))
	rep := r.report()
	if code != 1 || rep["ok"] != false {
		t.Fatalf("exit %d ok=%v", code, rep["ok"])
	}
	all := steps(rep)
	at := -1
	for i, s := range all {
		if s["required"] == true && s["verdict"] != "verified" {
			at = i
			break
		}
	}
	if at < 0 || all[at]["name"] != "read outputs == true" || all[at]["verdict"] != "mismatch" {
		t.Fatalf("first failed required step: %v", all[at])
	}
	skipped := false
	for _, s := range all[at+1:] {
		skipped = skipped || s["verdict"] == "skipped"
	}
	if !skipped || r.fake.Has(Scratch) || all[len(all)-1]["name"] != "cleanup" {
		t.Errorf("skipped later=%v present=%v last=%v", skipped, r.fake.Has(Scratch), all[len(all)-1]["name"])
	}
}

func TestAServerThatCannotStartIsANonzeroExit(t *testing.T) {
	r := newRunner(t)
	code, out, errOut := r.run(r.argv(true, "--insecure-tls", "Nope"))
	if code != 1 || r.report()["ok"] != false || r.fake.Requests() != 0 {
		t.Errorf("exit %d requests=%d\n%s%s", code, r.fake.Requests(), out, errOut)
	}
}

func TestNoCredentialValueInStdoutStderrOrReport(t *testing.T) {
	r := newRunner(t)
	for _, argv := range [][]string{r.argv(true), r.argv(false)} {
		_, out, errOut := r.run(argv)
		blob := out + errOut
		if data, err := os.ReadFile(r.reportPath); err == nil {
			blob += string(data)
		}
		if strings.Contains(blob, password) || strings.Contains(blob, user) {
			t.Errorf("%v leaks a credential", argv)
		}
	}
}

func TestAFailedLoginDoesNotLeakThePassword(t *testing.T) {
	r := newRunner(t)
	r.setEnv("MCP_N4_PASSWORD=" + password + "-wrong")
	code, out, errOut := r.run(r.argv(true))
	data, _ := os.ReadFile(r.reportPath)
	if code != 1 || strings.Contains(out+errOut+string(data), password) || r.fake.Has(Scratch) {
		t.Errorf("exit %d present=%v\n%s", code, r.fake.Has(Scratch), out)
	}
}
