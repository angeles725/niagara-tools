package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// gradlewEnv makes the stub gradlew log the two variables the mirror isolation sets, once per call.
func gradlewEnv(f *fixture, home string) {
	f.write(f.dir+"/do/gradlew", `echo "JTO=[${JAVA_TOOL_OPTIONS-}] GUH=[${GRADLE_USER_HOME-}]" >> "`+f.dir+`/gradle.env"`+"\n"+installing(home, "", "rt", "ux"))
}

// A mirror NIAGARA_HOME gets the isolation build.sh applies (gradleenv.Isolate): user.home in the mirror for
// every JVM, GRADLE_USER_HOME and the real maven repository pinned, and --no-daemon on every gradlew call,
// slotomatic included (retro 2026-11-15-coordination-stretch-deltas D3).
func TestAMirrorBuildIsIsolatedAsBuildShIsolatesIt(t *testing.T) {
	f := newFixture(t)
	withHome(t, f, true)
	home := f.dir + "/home"
	f.remove(home + "/.mirror-user-home") // withHome made it: this run must
	gradlewEnv(f, home)
	t.Setenv("HOME", f.dir+"/real-home")
	f.want(f.run("--no-backup", "--with-slotomatic"), ExitOK,
		"[ng-deploy] mirror niagara_home: user.home isolated to "+home+"/.mirror-user-home (GRADLE_USER_HOME="+f.dir+"/gradle-user-home, --no-daemon) so the signing plugin cannot read the real ~/.tridium keystore\n")
	want := "JTO=[-Duser.home=" + home + "/.mirror-user-home -Dmaven.repo.local=" + f.dir + "/real-home/.m2/repository] GUH=[" + f.dir + "/gradle-user-home]\n"
	if got := f.read(f.dir + "/gradle.env"); got != want+want {
		t.Errorf("gradlew saw:\n%swant twice:\n%s", got, want)
	}
	for _, call := range strings.Split(strings.TrimSpace(f.calls()), "\n") {
		if strings.HasPrefix(call, "gradlew ") && !strings.HasSuffix(call, " <--no-daemon>") {
			t.Errorf("gradlew call without --no-daemon: %s", call)
		}
	}
	if !f.exists(home + "/.mirror-user-home") {
		t.Error("the mirror user home was not made")
	}
}

// A home that is not there, or a real one opted in with --install-into-home, is not isolated: the gradlew
// calls and the environment stay as they were.
func TestAHomeThatIsNotAMirrorIsNotIsolated(t *testing.T) {
	f := newFixture(t) // NIAGARA_HOME names no directory
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK)
	if strings.Contains(f.calls(), "--no-daemon") || strings.Contains(f.out, "isolated") {
		t.Errorf("stdout:\n%scalls:\n%s", f.out, f.calls())
	}
}

// Whitespace in a value JAVA_TOOL_OPTIONS would carry is refused before gradle runs (exit 10).
func TestAMirrorWithWhitespaceIsRefusedBeforeGradle(t *testing.T) {
	f := newFixture(t)
	withHome(t, f, true)
	t.Setenv("HOME", f.dir+"/real home")
	code := f.run("--no-backup")
	if code != ExitEnv || !strings.Contains(f.err, "HOME contains whitespace: "+f.dir+"/real home") || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

// --gradle-prop k=v is forwarded to every gradlew call as -Pk=v, as build.sh forwards it, and the gate
// expects the niagaraSigningAlias it names; one that is not k=v is exit 10.
func TestGradlePropIsForwardedAsBuildShForwardsIt(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--mode", "C", "--gradle-prop", "niagaraSigningAlias=propsigner", "--gradle-prop", "niagaraSigningProfile=/p/x.xml"), ExitOK)
	if !strings.Contains(f.calls(), gradleProps+" <-PniagaraSigningAlias=propsigner> <-PniagaraSigningProfile=/p/x.xml> <:demo-rt:clean>") ||
		!strings.Contains(f.calls(), "<--expected-signer> <propsigner>") {
		t.Errorf("calls:\n%s", f.calls())
	}
	for _, bad := range [][]string{{"--gradle-prop", "nokey"}, {"--gradle-prop", "=v"}, {"--gradle-prop"}} {
		f = newFixture(t)
		code := f.run(bad...)
		if code != ExitEnv || !strings.Contains(f.err, "--gradle-prop needs k=v") || f.calls() != "" {
			t.Errorf("%v: exit %d, stderr %q", bad, code, f.err)
		}
	}
}

// The isolation goes to the commands the deploy starts, never into this process: after a mirror run the
// process environment is as it was, and a second run in the same process hands gradle the same values, not
// a second -Duser.home (review R3-env-leak).
func TestTheIsolationDoesNotLeakIntoTheProcess(t *testing.T) {
	f := newFixture(t)
	withHome(t, f, true)
	gradlewEnv(f, f.dir+"/home")
	t.Setenv("HOME", f.dir+"/real-home")
	before := os.Getenv("JAVA_TOOL_OPTIONS") + "|" + os.Getenv("GRADLE_USER_HOME")
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
	first := f.read(f.dir + "/gradle.env")
	if after := os.Getenv("JAVA_TOOL_OPTIONS") + "|" + os.Getenv("GRADLE_USER_HOME"); after != before {
		t.Errorf("the process environment changed: %q, was %q", after, before)
	}
	os.Remove(f.dir + "/gradle.env")
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
	if second := f.read(f.dir + "/gradle.env"); second != first || strings.Count(second, "-Duser.home=") != 1 {
		t.Errorf("second run:\n%sfirst:\n%s", second, first)
	}
}

// With two --gradle-prop niagaraSigningAlias, gradle gets both in order and the gate expects the last one,
// as gradle resolves a repeated -P (review R3-last-alias-untested).
func TestTheGateExpectsTheLastSigningAliasGiven(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--mode", "C", "--gradle-prop", "niagaraSigningAlias=first", "--gradle-prop", "niagaraSigningAlias=second"), ExitOK)
	if !strings.Contains(f.calls(), "<-PniagaraSigningAlias=first> <-PniagaraSigningAlias=second>") || !strings.Contains(f.calls(), "<--expected-signer> <second>") {
		t.Errorf("calls:\n%s", f.calls())
	}
}
