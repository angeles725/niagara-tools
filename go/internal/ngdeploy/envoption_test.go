package ngdeploy

import (
	"path/filepath"
	"strings"
	"testing"
)

// The option globals an env file sets (envload_test.go has how the file is read).
// Kept from the bash tool: a key of the env file named like an option global sets that option when no flag
// did (envflags_test.go and envguard_test.go have the keys it may not set by itself, and the flag that wins).
func TestAnEnvFileKeyNamedLikeAnOptionSetsIt(t *testing.T) {
	f := newFixture(t)
	f.set("MODE", "C") // MODE=B, the mode without the gate, is refused (envguard_test.go)
	f.set("NO_DEPLOY", "1")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] build: mode C — :demo-rt:clean :demo-rt:jar\n", "[ng-deploy] no-deploy: jars in build/libs/\n", "err:"+warnNoBackup)

	f = newFixture(t)
	f.baseline()
	t.Setenv("FAKE_SR_EXIT", "1")
	t.Setenv("FAKE_VM_EXIT", "1")
	f.set("ACCEPT_LOSSY", "1")
	f.set("NO_GATE", "1")
	f.set("WITH_SLOTOMATIC", "1")
	f.set("VERIFY_MODULE_BIN", f.dir+"/absent")
	f.want(f.run("--no-backup", "--no-gate", "--mode", "C"), ExitOK, "[ng-deploy] slotomatic ok\n", "[ng-deploy] copy ok\n", "err:[ng-deploy] WARN schema-risk LOSSY accepted (--accept-lossy)\n")

	f = newFixture(t)
	f.set("FULL_BACKUP", "1")
	f.set("KEEP_N", "1")
	f.set("NO_GATE", "") // an empty value is off, as bash's [[ "" -eq 1 ]] is
	f.deployed()
	oldBackups(f)
	f.want(f.run("--no-deploy"), ExitOK)
	kept, _ := filepath.Glob(f.work + "/_backups/demo-pre-*")
	if !strings.Contains(f.calls(), "<-C> <"+f.dir+"> <modules>\n") || len(kept) != 1 {
		t.Errorf("FULL_BACKUP=1 and KEEP_N=1 of the env file: %d backups kept, calls:\n%s", len(kept), f.calls())
	}
}

// Declared difference: bash stopped with exit 1 and "unbound variable" when it reached the test.
func TestAnOptionValueThatIsNotAWholeNumberIsRefusedBeforeAnythingRuns(t *testing.T) {
	for _, key := range []string{"NO_BACKUP", "NO_DEPLOY", "NO_GATE", "WITH_SLOTOMATIC", "STRICT_SLOTOMATIC", "FIRST_DEPLOY", "ACCEPT_LOSSY", "FULL_BACKUP"} {
		f := newFixture(t)
		f.set(key, "yes")
		code := f.run("--mode", "C")
		if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR "+key+" must be 0 or 1\n" || f.calls() != "" {
			t.Errorf("%s=yes: exit %d, stdout %q, stderr %q, calls %q", key, code, f.out, f.err, f.calls())
		}
	}
	f := newFixture(t)
	t.Setenv("FULL_BACKUP", "true")
	if code := f.run(); code != ExitEnv || f.err != "[ng-deploy] ERROR FULL_BACKUP must be 0 or 1\n" {
		t.Errorf("FULL_BACKUP=true in the environment: exit %d, stderr %q", code, f.err)
	}
}

// Declared difference: bash deployed nothing and still ended "copy ok" with exit 0.
func TestAnInvalidModeFromTheEnvFileIsRefused(t *testing.T) {
	f := newFixture(t)
	f.set("MODE", "Z")
	code := f.run("--no-backup")
	if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR MODE set by the env file is not A, B, C or D\n" || f.calls() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
}

// Kept from the bash tool: a variable of the environment named like an option global is handed on with
// the deploy's value, and the launcher's own variables are not handed on at all.
func TestCommandsSeeOptionGlobalsWithTheDeployValue(t *testing.T) {
	f := newFixture(t)
	f.set("FIRST_DEPLOY", "-")
	t.Setenv("FIRST_DEPLOY", "1")
	t.Setenv("MODE", "ambient")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "B"), ExitOK)
	if !strings.Contains(f.envLog(), "gradlew MODE=[B] FD=[0] FILEKEY=[unset] MN=[demo] SELF=[unset]\n") {
		t.Errorf("gradlew saw:\n%s", f.envLog())
	}
}
