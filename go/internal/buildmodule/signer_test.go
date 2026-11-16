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
