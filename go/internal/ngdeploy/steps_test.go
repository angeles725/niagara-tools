package ngdeploy

import (
	"strings"
	"testing"
)

// step runs one step as the sourced launcher does: its name in StepEnv, its arguments as the command line.
func (f *fixture) step(name string, args ...string) int {
	f.t.Setenv(StepEnv, name)
	f.noEnvFile = true
	return f.run(args...)
}

func TestSourcedStepsRunOnePhaseFromTheEnvironment(t *testing.T) {
	f := newFixture(t)
	f.want(f.step("verify_jar", "fake.jar", "9"), ExitOK, "[ng-deploy] verify ok (fake.jar: 9/9)\n")
	f.want(f.step("verify_jar", "fake.jar", "3"), ExitVerify, "err:[ng-deploy] ERROR verify failed: expected 3 <type entries in fake.jar, found 9\n")
	if code := f.step("verify_cachebuster", "fake-ux.jar"); code != ExitOK || f.out != "" {
		t.Errorf("verify_cachebuster without BUILD_ID: exit %d, stdout %q", code, f.out)
	}
	t.Setenv("BUILD_ID", "abc")
	t.Setenv("FAKE_INDEX", "?v=WRONG")
	f.want(f.step("verify_cachebuster", "fake-ux.jar"), ExitVerify, "err:index.html in fake-ux.jar does not contain ?v=abc (BUILD_ID check)\n")
	t.Setenv("FAKE_INDEX", "?v=abc")
	f.want(f.step("verify_cachebuster", "fake-ux.jar"), ExitOK, "[ng-deploy] verify ok (cache-buster ?v=abc found)\n")
}

func TestTheSourcedBackupStepReadsItsSettingsFromTheEnvironment(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	t.Setenv("MODULE_NAME", "demo")
	t.Setenv("STATION_MODULES_DIR", f.modules)
	f.want(f.step("backup"), ExitOK, "[ng-deploy] backup ok: _backups/demo-pre-")
	if !strings.Contains(f.calls(), "<-C> <"+f.modules+"> <demo-rt.jar> <demo-ux.jar>\n") || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("calls:\n%s", f.calls())
	}
	f.reset()
	t.Setenv("FULL_BACKUP", "1")
	f.want(f.step("backup"), ExitOK)
	if !strings.Contains(f.calls(), "<-C> <"+f.dir+"> <modules>\n") {
		t.Errorf("calls:\n%s", f.calls())
	}
	t.Setenv("MODULE_NAME", "")
	f.want(f.step("backup"), ExitEnv, "err:[ng-deploy] ERROR MODULE_NAME is required but not set\n")
}

func TestAnUnknownSourcedStepIsRefused(t *testing.T) {
	f := newFixture(t)
	if code := f.step("deploy_everything"); code != ExitEnv || f.err != "[ng-deploy] ERROR unknown step: deploy_everything\n" || f.calls() != "" {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}
