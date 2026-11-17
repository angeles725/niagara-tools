package ngdeploy

import (
	"strings"
	"testing"
)

// MODE=B (a mode without the gate) and VERIFY_MODULE_BIN (any executable stands in for the gate) are the
// other two ways an env file could skip the verify-module gate: each is refused with file and line before
// anything runs, as NO_GATE=1 is, unless the command line says the same thing (retro go-ng-deploy-port Δ12).
func TestAnEnvFileMayNotSkipTheGateWithModeBOrAnotherGate(t *testing.T) {
	for _, tc := range []struct{ key, value, label, flag string }{
		{"MODE", "B", "MODE=B", "--mode B"},
		{"VERIFY_MODULE_BIN", "/bin/true", "VERIFY_MODULE_BIN", "--no-gate"},
	} {
		f := newFixture(t)
		f.deployed()
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		t.Setenv("FAKE_VM_EXIT", "1") // the gate fails if it runs
		f.set(tc.key, tc.value)
		code := f.run("--no-backup")
		want := "[ng-deploy] ERROR env file " + f.env + ": line 10: " + tc.label + " would switch the verify-module gate off, which an env file may not do: give " + tc.flag + " on the command line\n"
		if code != ExitEnv || f.out != "" || f.err != want || f.calls() != "" {
			t.Errorf("%s=%s: exit %d, stdout %q, stderr %q, calls %q", tc.key, tc.value, code, f.out, f.err, f.calls())
		}
	}
}

// A CRLF env file is refused the same way: the carriage return is not a way around the MODE=B check.
func TestACRLFEnvFileMayNotSkipTheGateWithModeB(t *testing.T) {
	f := newFixture(t)
	f.write(f.env, strings.ReplaceAll(f.read(f.env)+"MODE=B\n", "\n", "\r\n"))
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR env file " + f.env + ": line 10: MODE=B would switch the verify-module gate off, which an env file may not do: give --mode B on the command line\n"
	if code != ExitEnv || f.err != want || f.calls() != "" {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

// What stays allowed: a MODE that keeps the gate, MODE=B when --mode is given (the flag wins over the file),
// VERIFY_MODULE_BIN with --no-gate (the gate does not run) or from the environment of the command.
func TestTheGateKeysStillWorkWhereTheyCannotSkipTheGate(t *testing.T) {
	newFixture := func(t *testing.T) *fixture { // a deploy that raises the version
		f := newFixture(t)
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		return f
	}
	f := newFixture(t)
	f.set("MODE", "C")
	f.set("NO_DEPLOY", "1")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] build: mode C — :demo-rt:clean :demo-rt:jar\n")

	f = newFixture(t)
	f.set("MODE", "B")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "B"), ExitOK, "[ng-deploy] build: mode B — :demo-ux:clean :demo-ux:jar\n")

	f = newFixture(t)
	f.set("MODE", "B")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK, "[ng-deploy] build: mode C — ")

	f = newFixture(t)
	f.deployed()
	f.set("VERIFY_MODULE_BIN", f.dir+"/absent")
	f.want(f.run("--no-backup", "--no-gate", "--mode", "C"), ExitOK, "[ng-deploy] copy ok\n")

	f = newFixture(t)
	f.deployed()
	t.Setenv("VERIFY_MODULE_BIN", f.dir+"/absent") // the environment of the command, not the env file
	if code := f.run("--no-backup", "--mode", "C"); code != ExitVerify || !strings.Contains(f.err, "absent") {
		t.Errorf("VERIFY_MODULE_BIN from the environment: exit %d, stderr %q", code, f.err)
	}
}
