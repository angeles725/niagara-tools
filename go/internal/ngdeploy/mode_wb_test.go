package ngdeploy

import (
	"strings"
	"testing"
)

// Retro build-kit-bench-session Δ5: mode D deploys the -rt and the -wb jar of a module with a workbench part (the driver
// scaffold builds those two): it builds, verifies, backs up and copies both, with the type count of the -wb jar in
// EXPECTED_WB_TYPES, and no -ux jar is built or touched.
func TestModeDDeploysTheRtAndTheWbJar(t *testing.T) {
	f := newFixture(t)
	f.write(f.builtJar("wb"), "wbjar")
	f.set("EXPECTED_WB_TYPES", "9")
	f.set("FIRST_DEPLOY", "1")
	if code := f.run("--mode", "D"); code != ExitOK {
		t.Fatalf("exit %d\nstdout:\n%sstderr:\n%s", code, f.out, f.err)
	}
	if f.read(f.deployedJar("rt")) != "rtjar" || f.read(f.deployedJar("wb")) != "wbjar" || f.exists(f.deployedJar("ux")) {
		t.Errorf("station holds rt %q wb %q ux present %v", f.read(f.deployedJar("rt")), f.read(f.deployedJar("wb")), f.exists(f.deployedJar("ux")))
	}
	if calls := f.calls(); !strings.Contains(calls, "<:demo-rt:clean> <:demo-wb:clean> <:demo-rt:jar> <:demo-wb:jar>") || strings.Contains(calls, "demo-ux") {
		t.Errorf("gradle calls:\n%s", calls)
	}
	if !strings.Contains(f.out, "demo-wb.jar") {
		t.Errorf("stdout does not name the -wb jar:\n%s", f.out)
	}
}

// Mode D needs the type count of the -wb jar like A needs the -ux one; a count that is not the jar's is exit 50.
func TestModeDNeedsAndChecksTheWbTypeCount(t *testing.T) {
	f := newFixture(t)
	f.write(f.builtJar("wb"), "wbjar")
	if code := f.run("--mode", "D"); code != ExitEnv || !strings.Contains(f.err, "EXPECTED_WB_TYPES is required for mode D") || f.calls() != "" {
		t.Errorf("no count: exit %d, stderr %q", code, f.err)
	}
	f.set("EXPECTED_WB_TYPES", "x")
	if code := f.run("--mode", "D"); code != ExitEnv || !strings.Contains(f.err, "EXPECTED_WB_TYPES") {
		t.Errorf("not a number: exit %d, stderr %q", code, f.err)
	}
}

// PRESERVE: the other modes still say which modes there are, and a -wb jar's .ng-deploy-old, left by a mode D deploy
// that did not finish, does not stop a deploy of a mode that never touches the -wb jar.
func TestTheOtherModesAreAsTheyWere(t *testing.T) {
	f := newFixture(t)
	f.write(f.deployedJar("wb")+".ng-deploy-old", "previous wb")
	if code := f.run("--no-backup", "--mode", "C"); code != ExitOK {
		t.Errorf("mode C beside a -wb .ng-deploy-old: exit %d, stderr %q", code, f.err)
	}
	if code := f.run("--mode", "E"); code != ExitEnv || f.err != "[ng-deploy] ERROR invalid --mode 'E'; valid values are A, B, C, D\n" {
		t.Errorf("mode E: exit %d, stderr %q", code, f.err)
	}
}
