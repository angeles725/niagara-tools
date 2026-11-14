package ngdeploy

import (
	"strings"
	"testing"
)

func TestEnvFileValuesWinOverTheEnvironmentAndReachEveryCommand(t *testing.T) {
	f := newFixture(t)
	t.Setenv("MODULE_NAME", "ambient")
	f.write(f.env, f.read(f.env)+"\n# a comment\nexport FILEKEY='single $quoted'\nBUILD_ID=\"b 42\"   # trailing\n")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK, "[ng-deploy] build: mode C — :demo-rt:clean\n:demo-rt:jar\n")
	if !strings.Contains(f.envLog(), "gradlew MODE=[unset] FD=[1] FILEKEY=[single $quoted] MN=[demo] SELF=[unset]\n") {
		t.Errorf("gradlew did not see the env file:\n%s", f.envLog())
	}
}

func TestAMissingEnvFileIsToleratedAndTheEnvironmentIsUsed(t *testing.T) {
	f := newFixture(t)
	for k, v := range map[string]string{"MODULE_NAME": "demo", "GRADLEW_PATH": f.src + "/gradlew", "NIAGARA_HOME": "/n", "NIAGARA_USER_HOME": "/u",
		"JAVA_HOME": "/j", "STATION_MODULES_DIR": f.modules, "EXPECTED_RT_TYPES": "9"} {
		t.Setenv(k, v)
	}
	f.noEnvFile = true
	for _, file := range []string{f.dir + "/absent", f.dir + "/absent/below", f.env + "/below", "/dev/null"} {
		f.want(f.run("--env-file", file, "--no-backup", "--no-deploy", "--mode", "C"), ExitOK, "[ng-deploy] build ok\n")
		if f.err != warnNoBackup {
			t.Errorf("%s: stderr %q", file, f.err)
		}
	}
}

// A directory is not "no env file": the operator named the wrong path, and the deploy would run on the
// ambient environment alone.
func TestAnEnvFileThatIsADirectoryIsRefused(t *testing.T) {
	f := newFixture(t)
	for k, v := range map[string]string{"MODULE_NAME": "demo", "GRADLEW_PATH": f.src + "/gradlew", "NIAGARA_HOME": "/n", "NIAGARA_USER_HOME": "/u",
		"JAVA_HOME": "/j", "STATION_MODULES_DIR": f.modules, "EXPECTED_RT_TYPES": "9"} {
		t.Setenv(k, v)
	}
	f.noEnvFile = true
	code := f.run("--env-file", f.modules, "--no-backup", "--no-deploy", "--mode", "C")
	if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR env file "+f.modules+" is a directory\n" || f.calls() != "" {
		t.Errorf("--env-file <directory>: exit %d, stdout %q, stderr %q, calls %q", code, f.out, f.err, f.calls())
	}
	f.mkdir(f.work + "/.env.local") // the default path is checked the same way
	code = f.run("--no-backup", "--no-deploy", "--mode", "C")
	if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR env file .env.local is a directory\n" || f.calls() != "" {
		t.Errorf(".env.local is a directory: exit %d, stdout %q, stderr %q, calls %q", code, f.out, f.err, f.calls())
	}
}

func TestTheDefaultEnvFileIsDotEnvLocalInTheWorkingDirectory(t *testing.T) {
	f := newFixture(t)
	f.write(f.work+"/.env.local", f.read(f.env))
	f.noEnvFile = true
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] no-deploy: jars in build/libs/\n")
}

// Declared difference from the bash tool, which sourced the file: anything but a plain assignment is refused.
func TestAnEnvFileLineThatIsNotAPlainAssignmentIsRefusedWithFileAndLine(t *testing.T) {
	for name, line := range map[string]string{"substitution": "TOKEN=hunter2$(date)", "default form": "X=${HOME:-hunter2}",
		"command": "echo hunter2", "another user's home": "X=~root/hunter2", "shell parameter": "X=hunter2$1"} {
		f := newFixture(t)
		f.write(f.env, f.read(f.env)+line+"\n")
		code := f.run("--no-backup")
		if code != ExitEnv || f.out != "" || !strings.HasPrefix(f.err, "[ng-deploy] ERROR env file "+f.env+": line 10: ") {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", name, code, f.out, f.err)
		}
		if strings.Contains(f.err, "hunter2") || f.calls() != "" {
			t.Errorf("%s: the refusal quoted the line or ran a command: %q %q", name, f.err, f.calls())
		}
	}
}

// The env file sees what it saw when bash sourced it: its own earlier lines, the option globals of the
// deploy (as the flags left them), the script's own variables and the environment.
func TestAnEnvFileExpandsPlainVariablesAsBashDid(t *testing.T) {
	f := newFixture(t)
	t.Setenv("HOME", "/hm")
	t.Setenv("AMBIENT", "from env")
	f.write(f.env, f.read(f.env)+"BASE=/n4\nNIAGARA_USER_HOME=\"${BASE}/user $AMBIENT\"\nNIAGARA_HOME=~/niagara\nJAVA_HOME='$BASE'\n"+
		"FILEKEY=$MODE:$KEEP_N:$FIRST_DEPLOY:${SCRIPT_DIR}\n")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C", "--keep", "7"), ExitOK)
	want := "gradlew <-Pniagara_home=/hm/niagara> <-Pniagara_user_home=/n4/user from env> <-Porg.gradle.java.installations.paths=$BASE> <:demo-rt:clean> <:demo-rt:jar>\n"
	if !strings.HasSuffix(f.calls(), want) {
		t.Errorf("calls:\n%s", f.calls())
	}
	if !strings.Contains(f.envLog(), "FILEKEY=[C:7:1:"+f.dir+"/repo/scripts]") {
		t.Errorf("gradlew saw:\n%s", f.envLog())
	}
}

// The bash tool sourced the file under set -u and stopped with exit 1 "unbound variable".
func TestAnEnvFileVariableThatIsNotSetIsRefusedNamingIt(t *testing.T) {
	f := newFixture(t)
	f.write(f.env, f.read(f.env)+"TOKEN=hunter2$NOT_SET_ANYWHERE\n")
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR env file " + f.env + ": line 10: NOT_SET_ANYWHERE is not set (the file is read as under bash's set -u: an unset variable stops the read)\n"
	if code != ExitEnv || f.out != "" || f.err != want || f.calls() != "" {
		t.Errorf("exit %d, stdout %q, stderr %q, calls %q", code, f.out, f.err, f.calls())
	}
}

func TestAnEnvFileThatCannotBeReadIsRefused(t *testing.T) {
	f := newFixture(t)
	f.chmod(f.env, 0o000)
	code := f.run("--no-backup")
	if code != ExitEnv || f.err != "[ng-deploy] ERROR env file "+f.env+" cannot be read: Permission denied\n" || f.calls() != "" {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

func TestAnEnvFileMayNotAssignWhatTheDeployOwns(t *testing.T) {
	for _, key := range []string{"SCRIPT_DIR", "VERSION_FILE", "SCRIPT_VERSION", "IFS"} {
		f := newFixture(t)
		f.set(key, "x")
		code := f.run("--no-backup")
		if code != ExitEnv || f.err != "[ng-deploy] ERROR env file "+f.env+": line 10: "+key+" is set by ng-deploy and cannot be assigned\n" || f.calls() != "" {
			t.Errorf("%s: exit %d, stderr %q", key, code, f.err)
		}
	}
}

// Kept from the bash tool: the env file is read after the flags, so a key named like an option global
// replaces what the flag set.
func TestAnEnvFileKeyNamedLikeAnOptionReplacesTheFlag(t *testing.T) {
	f := newFixture(t)
	f.set("MODE", "B")
	f.set("NO_BACKUP", "1")
	f.set("NO_DEPLOY", "1")
	f.want(f.run("--mode", "C"), ExitOK, "[ng-deploy] build: mode B — :demo-ux:clean\n:demo-ux:jar\n", "[ng-deploy] no-deploy: jars in build/libs/\n", "err:"+warnNoBackup)

	f = newFixture(t)
	f.baseline()
	t.Setenv("FAKE_SR_EXIT", "1")
	t.Setenv("FAKE_VM_EXIT", "1")
	f.set("ACCEPT_LOSSY", "1")
	f.set("NO_GATE", "1")
	f.set("WITH_SLOTOMATIC", "1")
	f.set("VERIFY_MODULE_BIN", f.dir+"/absent")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] slotomatic ok\n", "[ng-deploy] copy ok\n", "err:[ng-deploy] WARN schema-risk LOSSY accepted (--accept-lossy)\n")

	f = newFixture(t)
	f.set("FULL_BACKUP", "0")
	f.set("NO_GATE", "") // an empty value is off, as bash's [[ "" -eq 1 ]] is
	f.deployed()
	f.want(f.run("--full-backup", "--no-deploy"), ExitOK)
	if !strings.Contains(f.calls(), "tar <-czf> <_backups/demo-pre-") || !strings.Contains(f.calls(), "<-C> <"+f.modules+"> <demo-rt.jar> <demo-ux.jar>\n") {
		t.Errorf("FULL_BACKUP=0 in the env file did not win over --full-backup:\n%s", f.calls())
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
	if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR MODE set by the env file is not A, B or C\n" || f.calls() != "" {
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
