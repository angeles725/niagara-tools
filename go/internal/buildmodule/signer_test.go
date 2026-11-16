package buildmodule

import (
	"strings"
	"testing"
)

// noSignerWarn is the one stderr line of a build that has no signer to expect.
const noSignerWarn = "build.sh: WARN — no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature: " +
	"the jar is signed with the signing plugin's default development signer unless the gradle root names another, " +
	"and a station's Software Manager shows that one as self-signed (set it: --gradle-prop niagaraSigningAlias=<alias>, or gradle.properties)\n"

// TestExpectedSignerSource: the gate gets --expected-signer with the configured niagaraSigningAlias (in
// gradle's precedence: --gradle-prop, ORG_GRADLE_PROJECT_, GRADLE_USER_HOME/gradle.properties, the gradle
// root's gradle.properties), else the one signer of the deployed jars the drift gate snapshots, else it gets
// no such option and the build WARNs once; the exit code is the same either way.
func TestExpectedSignerSource(t *testing.T) {
	for _, c := range []struct {
		name   string
		setup  func(f *fixture) []string // returns the flags
		signer string                    // "" for no option and the WARN
	}{
		{"--gradle-prop", func(f *fixture) []string { return []string{"--gradle-prop", "niagaraSigningAlias=acmesigner"} }, "acmesigner"},
		{"the last --gradle-prop wins", func(f *fixture) []string {
			return []string{"--gradle-prop", "niagaraSigningAlias=other", "--gradle-prop", "niagaraSigningAlias=acmesigner"}
		}, "acmesigner"},
		{"ORG_GRADLE_PROJECT_ environment", func(f *fixture) []string {
			f.env["ORG_GRADLE_PROJECT_niagaraSigningAlias"] = "acmesigner"
			return nil
		}, "acmesigner"},
		{"the gradle root's gradle.properties", func(f *fixture) []string {
			f.write(f.root+"/gradle.properties", "# niagaraSigningAlias=commented\nniagaraSigningAlias = acmesigner\n")
			return nil
		}, "acmesigner"},
		{"GRADLE_USER_HOME/gradle.properties comes before the root's", func(f *fixture) []string {
			f.env["GRADLE_USER_HOME"] = f.dir + "/guh"
			f.write(f.dir+"/guh/gradle.properties", "niagaraSigningAlias=acmesigner\n")
			f.write(f.root+"/gradle.properties", "niagaraSigningAlias=other\n")
			return nil
		}, "acmesigner"},
		{"the configured alias comes before the deployed jar's signer", func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "p", "NIAGARA4")
			return []string{"--gradle-prop", "niagaraSigningAlias=acmesigner"}
		}, "acmesigner"},
		{"the deployed jar's signer", func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "p", "ACMESIGN")
			f.jar(f.deployed("ux"), "1.0.0", "p") // an unsigned deployed jar names nobody
			return nil
		}, "ACMESIGN"},
		{"deployed jars of two signers give none", func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "p", "ACMESIGN")
			f.jar(f.deployed("ux"), "1.0.0", "p", "NIAGARA4")
			return nil
		}, ""},
		{"a deployed jar with two signers gives none", func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "p", "ACMESIGN", "NIAGARA4")
			return nil
		}, ""},
		{"--no-drift-check reads no deployed jar", func(f *fixture) []string {
			f.jar(f.deployed("rt"), "1.0.0", "p", "ACMESIGN")
			return []string{"--no-drift-check"}
		}, ""},
		{"neither", func(f *fixture) []string { return nil }, ""},
		{"a configured alias is passed as given, the development signer's name included", func(f *fixture) []string {
			return []string{"--gradle-prop", "niagaraSigningAlias=NIAGARA4"}
		}, "NIAGARA4"},
	} {
		f := newFixture(t)
		f.wants(c.name, f.build(c.setup(f)...), ExitOK)
		option, stderr := "", noSignerWarn
		if c.signer != "" {
			option, stderr = " <--expected-signer> <"+c.signer+">", ""
		}
		if want := "verify-module.sh <--src> <" + f.root + "/Foo>" + option + " <" + f.built("rt") + "> <" + f.built("ux") + ">"; f.call(gateScript) != want {
			t.Errorf("%s: gate call %q, want %q", c.name, f.call(gateScript), want)
		}
		if f.err != stderr {
			t.Errorf("%s: stderr %q, want %q", c.name, f.err, stderr)
		}
	}
}

// TestExpectedSignerKeepsTheExitCode: the WARN changes no exit code, and a gate that FAILs on the signer is
// the chain's gate failure; --expected-signer comes after --target-version.
func TestExpectedSignerKeepsTheExitCode(t *testing.T) {
	f := newFixture(t)
	f.does(gateScript, "exit 1")
	f.wants("no signer, the gate fails", f.build(), ExitGate, noSignerWarn+"build.sh: verify gate failed — do not deploy these jars\n")
	f = newFixture(t)
	f.does(gateScript, "exit 1")
	f.wants("a signer, the gate fails", f.build("--target-version", "4.14", "--gradle-prop", "niagaraSigningAlias=acmesigner"), ExitGate)
	if f.err != "build.sh: verify gate failed — do not deploy these jars\n" || !strings.Contains(f.call(gateScript), " <--target-version> <4.14> <--expected-signer> <acmesigner> <") {
		t.Errorf("stderr %q, gate call %q", f.err, f.call(gateScript))
	}
}

// devSignerWarn is the stderr line of a build whose deployed jar carries the default development signer.
const devSignerWarn = "build.sh: WARN — no niagaraSigningAlias is configured and the deployed jar itself carries the signing plugin's default development signer (NIAGARA4), " +
	"which is never taken as the expected signer, so the gate accepts any signature: " +
	"a station's Software Manager shows that signer as self-signed (set it: --gradle-prop niagaraSigningAlias=<alias>, or gradle.properties)\n"

// TestDeployedDevelopmentSignerIsNeverExpected: a deployed jar signed only by the signing plugin's default
// development signer gives the gate no --expected-signer, and the WARN says the deployed jar carries it.
func TestDeployedDevelopmentSignerIsNeverExpected(t *testing.T) {
	f := newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "p", "NIAGARA4")
	f.jar(f.deployed("ux"), "1.0.0", "p") // unsigned: names nobody
	f.wants("development signer deployed", f.build(), ExitOK)
	if want := "verify-module.sh <--src> <" + f.root + "/Foo> <" + f.built("rt") + "> <" + f.built("ux") + ">"; f.call(gateScript) != want || f.err != devSignerWarn {
		t.Errorf("gate call %q, want %q\nstderr %q, want %q", f.call(gateScript), want, f.err, devSignerWarn)
	}
}

// TestGateFailRestoresTheDeployedJar: gradle's :jar step replaced the deployed jar before the gate ran, so a
// gate FAIL puts the snapshot back (exit code unchanged) and a bare re-run still expects the deployed
// signer and FAILs again; a gate PASS leaves the new jar in modules/. The gate stub FAILs exactly when it is
// asked for ACMESIGN, as the real gate does on a new jar signed with another key.
func TestGateFailRestoresTheDeployedJar(t *testing.T) {
	const failsOnAcme = `case " $* " in *" --expected-signer ACMESIGN "*) exit 1 ;; esac`
	f := newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "old", "ACMESIGN")
	f.jar(f.built("rt"), "1.0.1", "new", "NIAGARA4")
	before := f.read(f.deployed("rt"))
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`) // as the plugin's last :jar step
	f.does(gateScript, failsOnAcme)
	restored := "  Restored the previously-deployed Foo-rt.jar into " + f.home + "/modules.\nbuild.sh: verify gate failed — do not deploy these jars\n"
	for _, run := range []string{"first run", "bare re-run"} {
		f.wants(run, f.build("--profiles", "rt"), ExitGate)
		if f.err != restored {
			t.Errorf("%s: stderr %q, want %q", run, f.err, restored)
		}
		if f.read(f.deployed("rt")) != before {
			t.Errorf("%s: modules/ does not hold the pre-build jar", run)
		}
	}
	if n := strings.Count(f.calls(), "verify-module.sh <--src> <"+f.root+"/Foo> <--expected-signer> <ACMESIGN> <"); n != 2 {
		t.Errorf("the gate was asked for ACMESIGN %d times, want 2:\n%s", n, f.calls())
	}

	f = newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "old", "ACMESIGN")
	f.jar(f.built("rt"), "1.0.1", "new", "ACMESIGN")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
	f.wants("the gate passes", f.build("--profiles", "rt"), ExitOK)
	if f.err != "" || f.read(f.deployed("rt")) != f.read(f.built("rt")) {
		t.Errorf("a passing gate must leave the new jar in modules/: stderr %q", f.err)
	}

	f = newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "old", "ACMESIGN")
	f.jar(f.built("rt"), "1.0.1", "new", "NIAGARA4")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
	f.does(gateScript, "exit 1")
	f.wants("--no-drift-check has no snapshot", f.build("--profiles", "rt", "--no-drift-check"), ExitGate)
	if f.err != noSignerWarn+"build.sh: verify gate failed — do not deploy these jars\n" || f.read(f.deployed("rt")) != f.read(f.built("rt")) {
		t.Errorf("without a snapshot nothing is restored: stderr %q", f.err)
	}
}

// TestDeployedDevelopmentSignerInAnyCase: the development signer is recognised in the form the gate compares
// (verifymodule.SigfileName), so a deployed META-INF/niagara4.SF is never expected either.
func TestDeployedDevelopmentSignerInAnyCase(t *testing.T) {
	f := newFixture(t)
	f.jar(f.deployed("rt"), "1.0.0", "p", "niagara4")
	f.wants("lower-case development signer", f.build(), ExitOK)
	if strings.Contains(f.call(gateScript), "--expected-signer") || f.err != devSignerWarn {
		t.Errorf("gate call %q\nstderr %q, want %q", f.call(gateScript), f.err, devSignerWarn)
	}
}

// TestGateFailOnAFirstDeployRemovesTheNewJar: with no jar in modules/ before the build there is nothing to
// restore, so a gate FAIL removes the jar gradle just installed (exit code unchanged); a later run without
// the alias then has no deployed signer to take and WARNs instead of passing in silence. A gate PASS leaves
// the jar, and under --no-drift-check (no snapshot of what was deployed) nothing is removed.
func TestGateFailOnAFirstDeployRemovesTheNewJar(t *testing.T) {
	const failsOnAcme = `case " $* " in *" --expected-signer acmesigner "*) exit 1 ;; esac`
	alias := []string{"--profiles", "rt", "--gradle-prop", "niagaraSigningAlias=acmesigner"}
	first := func(signer string) *fixture {
		f := newFixture(t)
		f.jar(f.built("rt"), "1.0.0", "new", signer)
		f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`) // as the plugin's last :jar step
		f.does(gateScript, failsOnAcme)
		return f
	}
	f := first("OTHERKEY")
	f.wants("gate FAIL", f.build(alias...), ExitGate)
	removed := "  Removed the newly-installed Foo-rt.jar from " + f.home + "/modules (none was deployed before this build).\n" +
		"build.sh: verify gate failed — do not deploy these jars\n"
	if f.err != removed || exists(f.deployed("rt")) {
		t.Errorf("gate FAIL: stderr %q, want %q; modules/ holds the jar = %v", f.err, removed, exists(f.deployed("rt")))
	}
	f.write(f.dir+"/calls.log", "")
	f.wants("a later run without the alias", f.build("--profiles", "rt"), ExitOK)
	if strings.Contains(f.call(gateScript), "--expected-signer") || f.err != noSignerWarn {
		t.Errorf("a later run without the alias: gate call %q, stderr %q, want %q", f.call(gateScript), f.err, noSignerWarn)
	}

	f = first("ACMESIGN")
	f.does(gateScript, "exit 0")
	f.wants("gate PASS", f.build(alias...), ExitOK)
	if f.err != "" || f.read(f.deployed("rt")) != f.read(f.built("rt")) {
		t.Errorf("gate PASS must leave the new jar in modules/: stderr %q", f.err)
	}

	f = first("OTHERKEY")
	f.wants("--no-drift-check", f.build(append(alias, "--no-drift-check")...), ExitGate)
	if f.err != "build.sh: verify gate failed — do not deploy these jars\n" || !exists(f.deployed("rt")) {
		t.Errorf("--no-drift-check removes nothing: stderr %q", f.err)
	}
}
