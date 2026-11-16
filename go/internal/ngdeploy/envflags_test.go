package ngdeploy

import (
	"strings"
	"testing"
)

// A flag of the command line wins over the same option variable in the env file, whatever the file says:
// the deploy runs with the flag's value, and so does every command it starts.
func TestAFlagWinsOverTheSameOptionInTheEnvFile(t *testing.T) {
	f := newFixture(t)
	t.Setenv("MODE", "ambient") // exported, so the commands are handed the value the deploy settled on
	for key, value := range map[string]string{"MODE": "B", "KEEP_N": "5", "NO_DEPLOY": "0", "FULL_BACKUP": "0", "WITH_SLOTOMATIC": "0",
		"STRICT_SLOTOMATIC": "0", "ACCEPT_LOSSY": "0", "FIRST_DEPLOY": "0", "NO_BACKUP": "0", "NO_GATE": "0", "ENV_FILE": "/elsewhere"} {
		f.set(key, value)
	}
	f.deployed()
	oldBackups(f)
	f.want(f.run("--mode", "C", "--keep", "1", "--full-backup", "--no-deploy", "--with-slotomatic"), ExitOK,
		"[ng-deploy] slotomatic ok\n[ng-deploy] build: mode C — :demo-rt:clean :demo-rt:jar\n[ng-deploy] build ok\n[ng-deploy] backup: ", "[ng-deploy] no-deploy: jars in build/libs/\n")
	kept := 0
	for _, day := range []string{"1", "2", "3", "4", "5"} {
		if f.exists(f.work + "/_backups/demo-pre-2026010" + day + "-000000.tar.gz") {
			kept++
		}
	}
	if f.err != "" || kept != 0 || !strings.Contains(f.calls(), "<-C> <"+f.dir+"> <modules>\n") || !strings.Contains(f.envLog(), "gradlew MODE=[C] FD=[0]") {
		t.Errorf("stderr %q, %d old backups kept, calls:\n%senvironment:\n%s", f.err, kept, f.calls(), f.envLog())
	}

	f = newFixture(t) // the switches that skip a step: the flag is what skips it, the file's 0 does not undo it
	f.baseline()
	t.Setenv("FAKE_SR_EXIT", "1")
	t.Setenv("FAKE_VM_EXIT", "1")
	for _, key := range []string{"NO_BACKUP", "NO_GATE", "ACCEPT_LOSSY", "STRICT_SLOTOMATIC"} {
		f.set(key, "0")
	}
	f.want(f.run("--no-backup", "--no-gate", "--accept-lossy", "--mode", "C"), ExitOK, "[ng-deploy] copy ok\n", "err:"+warnNoBackup)
	if strings.Contains(f.calls(), "vm.sh") || strings.Contains(f.calls(), "tar") {
		t.Errorf("a 0 in the env file undid a flag:\n%s", f.calls())
	}
}

// The env file cannot switch the verify-module gate or the backup off by itself: the key is refused with
// file and line, before anything runs, unless the operator gave the flag too. A value that leaves the step
// on is read as before.
func TestAnEnvFileMayNotSwitchTheGateOrTheBackupOff(t *testing.T) {
	for key, tc := range map[string]struct{ what, flag string }{
		"NO_GATE":   {"the verify-module gate", "--no-gate"},
		"NO_BACKUP": {"the backup", "--no-backup"},
	} {
		for _, value := range []string{"1", "01", `"1"`} {
			f := newFixture(t)
			f.deployed()
			f.set(key, value)
			code := f.run("--mode", "C")
			want := "[ng-deploy] ERROR env file " + f.env + ": line 10: " + key + " would switch " + tc.what + " off, which an env file may not do: give " + tc.flag + " on the command line\n"
			if code != ExitEnv || f.out != "" || f.err != want || f.calls() != "" {
				t.Errorf("%s=%s: exit %d, stdout %q, stderr %q, calls %q", key, value, code, f.out, f.err, f.calls())
			}
			f.reset()
			f.want(f.run("--mode", "C", "--no-deploy", tc.flag), ExitOK, "[ng-deploy] build ok\n") // the flag is given: the key says the same
		}
		for _, value := range []string{"0", "", "2"} {
			f := newFixture(t)
			f.deployed()
			f.set(key, value)
			f.want(f.run("--mode", "C", "--no-deploy"), ExitOK, "[ng-deploy] backup ok: ", "[ng-deploy] build ok\n")
			if f.err != "" {
				t.Errorf("%s=%s: stderr %q", key, value, f.err)
			}
		}
	}
}
