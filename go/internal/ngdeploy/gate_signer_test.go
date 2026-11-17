package ngdeploy

import (
	"strings"
	"testing"
)

const (
	remedySaid   = " (set it: ORG_GRADLE_PROJECT_niagaraSigningAlias=<alias> in the env file, or niagaraSigningAlias=<alias> in gradle.properties)\n"
	noSignerWarn = "[ng-deploy] WARN no niagaraSigningAlias is configured and no deployed jar gives a signer, so the gate accepts any signature: " +
		"the jar is signed with the signing plugin's default development signer unless the gradle root names another, " +
		"and a station's Software Manager shows that one as self-signed" + remedySaid
	devSignerWarn = "[ng-deploy] WARN no niagaraSigningAlias is configured and the deployed jar itself carries the signing plugin's default development signer (NIAGARA4), " +
		"which is never taken as the expected signer, so the gate accepts any signature: " +
		"a station's Software Manager shows that signer as self-signed" + remedySaid
)

// The deploy gate gets --expected-signer with the signer build.sh resolves: the configured
// niagaraSigningAlias in gradle's precedence (the ORG_GRADLE_PROJECT_ variable, GRADLE_USER_HOME's
// gradle.properties, the gradle root's), else the one signer of the module's jars that were in
// <STATION_MODULES_DIR> before the copy, else of those that were in <NIAGARA_HOME>/modules before the build,
// never the development signer. With none it gets no such option
// and the deploy prints the warning build.sh prints, after the gate's rows; the exit code is the same.
func TestTheGateIsToldTheSignerBuildShExpects(t *testing.T) {
	noAlias := func(f *fixture) { f.remove(f.src + "/gradle.properties") }
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		signer  string // "" for no option
		warn    string
	}{
		{"the gradle root's gradle.properties", func(*fixture) {}, "acmesigner", ""},
		{"the ORG_GRADLE_PROJECT_ variable of the env file comes first", func(f *fixture) { f.set("ORG_GRADLE_PROJECT_niagaraSigningAlias", "fromenv") }, "fromenv", ""},
		{"GRADLE_USER_HOME/gradle.properties comes before the root's", func(f *fixture) {
			f.write(f.dir+"/gradle-user-home/gradle.properties", "niagaraSigningAlias=fromuserhome\n")
		}, "fromuserhome", ""},
		{"the configured alias comes before the deployed jar's signer", func(f *fixture) {
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("OTHER"))
		}, "acmesigner", ""},
		{"no alias: the signer of the jars that were in <NIAGARA_HOME>/modules", func(f *fixture) {
			noAlias(f)
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("ACMESIGN"))
		}, "ACMESIGN", ""},
		{"no alias: the development signer of a deployed jar is not expected", func(f *fixture) {
			noAlias(f)
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("NIAGARA4"))
		}, "", devSignerWarn},
		{"no alias: deployed jars of two signers give none", func(f *fixture) {
			noAlias(f)
			modules := withHome(f.t, f, true)
			f.zipJar(modules+"/demo-rt.jar", signedBy("ACMESIGN"))
			f.zipJar(modules+"/demo-ux.jar", signedBy("OTHER"))
		}, "", noSignerWarn},
		{"no alias and no deployed jar", noAlias, "", noSignerWarn},
		// bench Δ24: the jars the station holds give the signer too, and come before the home's.
		{"no alias: the signer of the jars the station holds", func(f *fixture) {
			noAlias(f)
			f.zipJar(f.deployedJar("rt"), signedBy("ACMESIGN"))
		}, "ACMESIGN", ""},
		{"no alias: the station's signer comes before the home's", func(f *fixture) {
			noAlias(f)
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("OTHER"))
			f.zipJar(f.deployedJar("rt"), signedBy("ACMESIGN"))
		}, "ACMESIGN", ""},
		{"no alias: the development signer of the station's jar is never expected", func(f *fixture) {
			noAlias(f)
			f.zipJar(f.deployedJar("rt"), signedBy("NIAGARA4"))
		}, "", devSignerWarn},
		{"no alias: station jars of two signers give none", func(f *fixture) {
			noAlias(f)
			f.zipJar(f.deployedJar("rt"), signedBy("ACMESIGN"))
			f.zipJar(f.deployedJar("ux"), signedBy("OTHER"))
		}, "", noSignerWarn},
		{"no alias: unsigned station jars leave the home's signer", func(f *fixture) {
			noAlias(f)
			f.zipJar(f.deployedJar("rt"), signedBy()) // a zip: a jar that is not one gives no signer at all
			f.zipJar(f.deployedJar("ux"), signedBy())
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("ACMESIGN"))
		}, "ACMESIGN", ""},
		{"no alias, the station is <NIAGARA_HOME>/modules: the jar there before the build, not the one it installed", func(f *fixture) {
			noAlias(f)
			modules := sameDir(f.t, f)
			f.zipJar(modules+"/demo-rt.jar", signedBy("ACMESIGN"))
			f.zipJar(modules+"/demo-ux.jar", signedBy())
			f.zipJar(f.builtJar("rt"), signedBy("OTHER")) // gradle installs it there before the gate runs
		}, "ACMESIGN", ""},
		{"no alias: a station jar of a profile outside the mode gives no signer", func(f *fixture) {
			noAlias(f)
			f.zipJar(f.modules+"/demo-wb.jar", signedBy("ACMESIGN"))
		}, "", noSignerWarn},
	} {
		f := newFixture(t)
		t.Setenv("FAKE_OLD_VERSION", "1.0") // a jar in the station is a deployed one: the build raises its version
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		tc.arrange(f)
		option := ""
		if tc.signer != "" {
			option = " <--expected-signer> <" + tc.signer + ">"
		}
		f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] gate ok\n", "[ng-deploy] copy ok\n")
		if want := "vm.sh <--src> <" + f.src + "/demo>" + option + " <" + f.builtJar("rt") + "> <" + f.builtJar("ux") + ">\n"; !strings.Contains(f.calls(), want) {
			t.Errorf("%s: the gate call is not %q:\n%s", tc.name, want, f.calls())
		}
		if f.err != warnNoBackup+tc.warn {
			t.Errorf("%s: stderr %q, want the warning %q", tc.name, f.err, tc.warn)
		}
	}
}

// The warning changes no exit code: a gate that fails is exit 50 with the warning before its ERROR line,
// and a run without the gate (--no-gate, mode B) resolves no signer and prints none.
func TestTheSignerWarningKeepsTheExitCode(t *testing.T) {
	f := newFixture(t)
	f.remove(f.src + "/gradle.properties")
	t.Setenv("FAKE_VM_EXIT", "1")
	code := f.run("--no-backup")
	if want := warnNoBackup + noSignerWarn + "[ng-deploy] ERROR verify-module gate failed; jars not fit to deploy (use --no-gate to override)\n"; code != ExitVerify || f.err != want {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	for _, args := range [][]string{{"--no-gate"}, {"--mode", "B"}} {
		f = newFixture(t)
		f.remove(f.src + "/gradle.properties")
		f.want(f.run(append([]string{"--no-backup"}, args...)...), ExitOK)
		if f.err != warnNoBackup {
			t.Errorf("%v: stderr %q", args, f.err)
		}
	}
}
