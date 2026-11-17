package ngdeploy

import (
	"strings"
	"testing"
)

func TestEnvFileValuesWinOverTheEnvironmentAndReachEveryCommand(t *testing.T) {
	f := newFixture(t)
	t.Setenv("MODULE_NAME", "ambient")
	f.write(f.env, f.read(f.env)+"\n# a comment\nexport FILEKEY='single $quoted'\nBUILD_ID=\"b 42\"   # trailing\n")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK, "[ng-deploy] build: mode C — :demo-rt:clean :demo-rt:jar\n")
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

// A CRLF env file used to put a carriage return at the end of every value: paths that do not exist, a
// module name no jar has. It is stripped, and one warning says so without quoting a value.
func TestACRLFEnvFileIsReadWithAWarning(t *testing.T) {
	f := newFixture(t)
	f.write(f.env, strings.ReplaceAll("# deploy settings\n\n"+f.read(f.env)+"FILEKEY='hunter2'\n", "\n", "\r\n"))
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK, "[ng-deploy] build: mode C — :demo-rt:clean :demo-rt:jar\n[ng-deploy] build ok\n")
	warning := "[ng-deploy] WARN env file " + f.env + ": a carriage return was stripped from the end of 12 line(s), the first at line 1 (CRLF line endings; save the file with LF)\n"
	if f.err != warning+warnNoBackup {
		t.Errorf("stderr %q", f.err)
	}
	if !strings.HasSuffix(f.calls(), "gradlew"+gradleProps+" <:demo-rt:clean> <:demo-rt:jar>\n") || !strings.Contains(f.envLog(), "FILEKEY=[hunter2] MN=[demo]") || strings.Contains(f.calls()+f.envLog(), "\r") {
		t.Errorf("a carriage return reached a command:\n%q\n%q", f.calls(), f.envLog())
	}
}

// An LF file is read as before, with no warning; a carriage return inside a value is the operator's.
func TestAnLFEnvFileIsReadWithoutAWarning(t *testing.T) {
	f := newFixture(t)
	f.write(f.env, f.read(f.env)+"FILEKEY='a\rb'\n")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "C"), ExitOK)
	if f.err != warnNoBackup || !strings.Contains(f.envLog(), "FILEKEY=[a\rb]") {
		t.Errorf("stderr %q, gradlew saw %q", f.err, f.envLog())
	}
}
