package buildmodule

import (
	"os"
	"strings"
	"testing"
)

// TestInputFaults: every input the chain cannot run with is exit 10 with its own message, before any command.
func TestInputFaults(t *testing.T) {
	cases := []struct {
		name  string
		setup func(f *fixture) (args []string)
		want  func(f *fixture) string // the whole stderr
	}{
		{"the module root does not exist",
			func(f *fixture) []string { return []string{f.dir + "/nope", "Foo", f.home} },
			func(f *fixture) string { return "build.sh: module root not found: " + f.dir + "/nope\n" }},
		{"gradlew is not executable",
			func(f *fixture) []string { os.Chmod(f.root+"/gradlew", 0o644); return nil },
			func(f *fixture) string {
				return "build.sh: " + f.root + "/gradlew exists but is not executable (a fresh VCS worktree of a repo that stores it 100644)\n" +
					"  fix: chmod +x " + f.root + "/gradlew\n"
			}},
		{"no gradlew here or above",
			func(f *fixture) []string { os.Remove(f.root + "/gradlew"); return nil },
			func(f *fixture) string {
				return "build.sh: no executable ./gradlew in " + f.root + " or any ancestor (the module needs the gradle wrapper)\n" +
					"  try: chmod +x " + f.root + "/gradlew\n"
			}},
		{"the repo root was passed: the gradle roots below are listed",
			func(f *fixture) []string {
				os.Remove(f.root + "/gradlew")
				for _, w := range []string{"GroupB", "GroupA/sub", "GroupC/build", "GroupC/.git", "GroupC/node_modules", "a/b/c"} {
					f.write(f.root+"/"+w+"/gradlew", "")
				}
				return nil
			},
			func(f *fixture) string {
				return "build.sh: no executable ./gradlew in " + f.root + " or any ancestor (the module needs the gradle wrapper)\n" +
					"  found gradle root(s) BELOW " + f.root + " — pass the GROUP dir that holds <MOD>/, not the repo root:\n" +
					"    " + f.root + "/GroupA/sub\n    " + f.root + "/GroupB\n  try: chmod +x " + f.root + "/gradlew\n"
			}},
		{"no JDK 8",
			func(f *fixture) []string { f.env["JAVA8"] = f.dir + "/no-jdk"; return nil },
			func(f *fixture) string {
				return "build.sh: Java 8 not found at " + f.dir + "/no-jdk — check 'ls /usr/lib/jvm' or set JAVA8\n"
			}},
		{"no niagara_home",
			func(f *fixture) []string { return []string{"--install-into-home", f.root, "Foo"} },
			func(f *fixture) string { return "build.sh: pass niagara_home (arg 3) or export niagara_home\n" }},
		{"not a niagara_home",
			func(f *fixture) []string { return []string{"--install-into-home", f.root, "Foo", f.dir} },
			func(f *fixture) string { return "build.sh: not a niagara_home (no etc/m2/repository): " + f.dir + "\n" }},
		{"a real niagara_home without the opt-in",
			func(f *fixture) []string { return []string{f.root, "Foo", f.home} },
			func(f *fixture) string {
				return "build.sh: " + f.home + " is a real niagara_home (no .niagara-mirror): gradle's :jar step would replace its modules/ jars\n" +
					"  build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home\n"
			}},
		{"the gate is not next to build.sh",
			func(f *fixture) []string { os.Remove(f.kit + "/verify-module.sh"); return nil },
			func(f *fixture) string {
				return "build.sh: gate not found next to this script: " + f.kit + "/verify-module.sh\n"
			}},
	}
	for _, c := range cases {
		f := newFixture(t)
		args := c.setup(f)
		if args == nil {
			args = []string{"--install-into-home", f.root, "Foo", f.home}
		}
		if code, want := f.run(args...), c.want(f); code != ExitEnv || f.err != want || f.out != "" {
			t.Errorf("%s: exit %d stdout %q\nstderr %q\nwant   %q", c.name, code, f.out, f.err, want)
		}
		if f.calls() != "" {
			t.Errorf("%s: ran %q", c.name, f.calls())
		}
	}
}

// TestGradleRootIsFoundAbove: a module dir below the gradle root builds there, with the module paths kept.
func TestGradleRootIsFoundAbove(t *testing.T) {
	f := newFixture(t)
	f.mkdir(f.root + "/group")
	if err := os.Rename(f.root+"/Foo", f.root+"/group/Foo"); err != nil {
		t.Fatal(err)
	}
	f.wants("ancestor gradlew", f.run("--install-into-home", f.root+"/group", "Foo", f.home), ExitOK)
	if !strings.Contains(f.read(f.dir+"/env.log"), "gradlew cwd="+f.root+" ") {
		t.Errorf("gradlew did not run in the gradle root: %s", f.read(f.dir+"/env.log"))
	}
	if want := "verify-module.sh <--src> <" + f.root + "/group/Foo> "; !strings.HasPrefix(f.call("verify-module.sh"), want) {
		t.Errorf("gate call %q, want prefix %q", f.call("verify-module.sh"), want)
	}
}

// TestNiagaraHomeComesFromTheEnvironment: an absent or empty third argument takes $niagara_home; a mirror
// needs no opt-in.
func TestNiagaraHomeComesFromTheEnvironment(t *testing.T) {
	for _, third := range [][]string{nil, {""}} {
		f := newFixture(t)
		f.write(f.home+"/"+mirrorMarker, "")
		f.env["niagara_home"] = f.home
		f.wants("niagara_home from the environment", f.run(append([]string{f.root, "Foo"}, third...)...), ExitOK)
		if !strings.Contains(f.call("gradlew"), "<-Pniagara_home="+f.home+">") {
			t.Errorf("gradlew call %q", f.call("gradlew"))
		}
	}
}

func TestToolbeltEnvIsNotHandedOn(t *testing.T) {
	f := newFixture(t)
	t.Setenv(ToolbeltEnv, f.kit)
	f.wants("build", f.build(), ExitOK)
	if env := f.read(f.dir + "/env.log"); strings.Count(env, "TB=[unset]") != 4 {
		t.Errorf("a command saw %s:\n%s", ToolbeltEnv, env)
	}
}

// TestHostToolFailuresEndTheChain: a temporary path mktemp cannot make ends the chain with mktemp's own row
// and status, before gradle (with the drift gate) or right after the build row (without it).
func TestHostToolFailuresEndTheChain(t *testing.T) {
	for _, flags := range [][]string{nil, {"--no-drift-check"}} {
		f := newFixture(t)
		t.Setenv("TMPDIR", f.dir+"/no-tmp")
		f.wants("no TMPDIR", f.build(flags...), 1, "mktemp: failed to create ")
		if f.call("gradlew") != "" {
			t.Errorf("%v: gradlew ran", flags)
		}
		if built := strings.Contains(f.out, "==> build"); built != (flags != nil) {
			t.Errorf("%v: build row printed = %v", flags, built)
		}
	}
}

func TestAModuleRootThatCannotBeEnteredIsRefused(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: every directory can be entered")
	}
	f := newFixture(t)
	f.mkdir(f.dir + "/locked")
	if err := os.Chmod(f.dir+"/locked", 0); err != nil {
		t.Fatal(err)
	}
	if code := f.run("--install-into-home", f.dir+"/locked", "Foo", f.home); code != ExitEnv || f.err != "build.sh: cannot resolve module root: \n" {
		t.Errorf("exit %d stderr %q", code, f.err)
	}
}
