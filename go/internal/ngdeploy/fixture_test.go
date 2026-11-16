package ngdeploy

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// The tests of this package are serial on purpose: each writes stub scripts and then runs them, and a
// parallel test that forks while a script is still open for writing makes its exec fail with "text file
// busy" (ETXTBSY). Do not add t.Parallel() here.
//
// fixture is a throwaway deploy: stub git, unzip and tar on PATH, a stub gradlew beside a built module
// "demo" (rt and ux jars), stub verify-module and schema-risk tools, an empty station modules directory, an
// env file and a launcher directory holding VERSION. Nothing real is built, and no host is contacted.
type fixture struct {
	t                                 *testing.T
	dir, env, modules, work, src, bin string
	noEnvFile                         bool
	out, err                          string // streams of the last run
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	d := t.TempDir()
	f := &fixture{t: t, dir: d, env: d + "/env", modules: d + "/modules", work: d + "/work", src: d + "/src", bin: d + "/bin"}
	keepEnvironment(t)
	for _, name := range fakeVariables {
		os.Unsetenv(name)
	}
	for name, do := range stubDo {
		path := f.bin + "/" + name
		switch name {
		case "gradlew":
			path = f.src + "/gradlew"
		case "vm.sh", "sr.sh":
			path = d + "/tools/" + name
		}
		f.write(path, strings.ReplaceAll(stub, "LOGDIR", d))
		f.chmod(path, 0o755)
		f.write(d+"/do/"+name, do)
	}
	for _, tool := range []string{"mkdir", "cp"} { // the host commands the deploy runs; everything else is a stub
		real, err := exec.LookPath(tool)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, f.bin+"/"+tool); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{f.modules, f.work, d + "/baseline"} {
		f.mkdir(dir)
	}
	f.write(f.builtJar("rt"), "rtjar")
	f.write(f.builtJar("ux"), "uxjar")
	f.write(d+"/repo/VERSION", "9.8.7\n")
	f.write(f.env, "MODULE_NAME=demo\nGRADLEW_PATH="+f.src+"/gradlew\nNIAGARA_HOME=\"/n4/niagara home\"\n"+
		"NIAGARA_USER_HOME=/n4/user\nJAVA_HOME=/n4/jdk8\nSTATION_MODULES_DIR="+f.modules+"\n"+
		"EXPECTED_RT_TYPES=9\nEXPECTED_UX_TYPES=9\nFIRST_DEPLOY=1\n")
	// The module names its signer as the kit's signing block reads it, and no gradle.properties of the host is read.
	f.write(f.src+"/gradle.properties", "niagaraSigningAlias=acmesigner\n")
	t.Setenv("GRADLE_USER_HOME", d+"/gradle-user-home")
	t.Setenv("PATH", f.bin)
	t.Setenv("VERIFY_MODULE_BIN", d+"/tools/vm.sh")
	t.Setenv("SCHEMA_RISK_BIN", d+"/tools/sr.sh")
	f.mkdir(d + "/repo/scripts")
	t.Setenv(ScriptEnv, d+"/repo/scripts/ng-deploy.sh")
	t.Chdir(f.work)
	return f
}

// keepEnvironment puts the process environment back when the test ends: a deploy exports its env file.
func keepEnvironment(t *testing.T) {
	saved := os.Environ()
	t.Cleanup(func() {
		os.Clearenv()
		for _, kv := range saved {
			k, v, _ := strings.Cut(kv, "=")
			os.Setenv(k, v)
		}
	})
}

// run is Main over args, after --env-file <the fixture's env file> unless noEnvFile.
func (f *fixture) run(args ...string) int {
	f.t.Helper()
	if !f.noEnvFile {
		args = append([]string{"--env-file", f.env}, args...)
	}
	os.Setenv(ScriptEnv, f.dir+"/repo/scripts/ng-deploy.sh") // Main takes it out of the environment
	var out, errs bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errs})
	f.out, f.err = out.String(), errs.String()
	return code
}

// builtJar is the jar the fake build "produced" for a profile; deployedJar is the one in the station.
func (f *fixture) builtJar(profile string) string {
	return f.src + "/demo/demo-" + profile + "/build/libs/demo-" + profile + ".jar"
}
func (f *fixture) deployedJar(profile string) string { return f.modules + "/demo-" + profile + ".jar" }

// deployed puts an older rt and ux jar in the station.
func (f *fixture) deployed() {
	f.write(f.deployedJar("rt"), "old-rt")
	f.write(f.deployedJar("ux"), "old-ux")
}

// set replaces (or with value "-", removes) a key of the env file.
func (f *fixture) set(key, value string) {
	f.t.Helper()
	var keep []string
	for _, line := range strings.Split(strings.TrimRight(f.read(f.env), "\n"), "\n") {
		if !strings.HasPrefix(line, key+"=") {
			keep = append(keep, line)
		}
	}
	if value != "-" {
		keep = append(keep, key+"="+value)
	}
	f.write(f.env, strings.Join(keep, "\n")+"\n")
}

// baseline makes the module a deployed one with a schema baseline.
func (f *fixture) baseline() {
	f.set("FIRST_DEPLOY", "-")
	f.set("SCHEMA_BASELINE_DIR", f.dir+"/baseline")
}

func (f *fixture) mkdir(dir string) {
	f.t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) write(path, content string) {
	f.t.Helper()
	f.mkdir(filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) remove(path string) {
	f.t.Helper()
	if err := os.Remove(path); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) chmod(path string, mode os.FileMode) {
	f.t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { os.Chmod(path, 0o755) })
}

// read is the text of a file, "" when there is none.
func (f *fixture) read(path string) string {
	b, _ := os.ReadFile(path)
	return string(b)
}

func (f *fixture) exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// calls is the log of every stub call, one "name <arg>..." line each; envLog the environment they saw.
func (f *fixture) calls() string  { return f.read(f.dir + "/calls.log") }
func (f *fixture) envLog() string { return f.read(f.dir + "/env.log") }

// reset forgets the calls of the runs so far.
func (f *fixture) reset() { os.Remove(f.dir + "/calls.log"); os.Remove(f.dir + "/env.log") }

// want fails the test unless the last run ended with code and its streams hold every given part
// ("err:" prefixed parts are looked for in stderr, the others in stdout).
func (f *fixture) want(got, code int, parts ...string) {
	f.t.Helper()
	if got != code {
		f.t.Errorf("exit %d, want %d\nstdout:\n%sstderr:\n%s", got, code, f.out, f.err)
	}
	for _, p := range parts {
		stream, name := f.out, "stdout"
		if rest, ok := strings.CutPrefix(p, "err:"); ok {
			stream, name, p = f.err, "stderr", rest
		}
		if !strings.Contains(stream, p) {
			f.t.Errorf("%s lacks %q:\n%s", name, p, stream)
		}
	}
}

// warnNoBackup is the stderr line of every run with --no-backup.
const warnNoBackup = "[ng-deploy] WARN backup skipped (--no-backup); ensure the module jars are committed to git for rollback.\n"

// gradleProps are the arguments every gradlew call starts with.
const gradleProps = " <-Pniagara_home=/n4/niagara home> <-Pniagara_user_home=/n4/user> <-Porg.gradle.java.installations.paths=/n4/jdk8>"
