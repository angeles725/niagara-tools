package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// signerGiven runs a deploy with no niagaraSigningAlias after arrange and returns the --expected-signer the
// gate got ("" for none) and stderr after the --no-backup warning.
func signerGiven(t *testing.T, arrange func(f *fixture)) (signer, warning string) {
	t.Helper()
	f := newFixture(t)
	t.Setenv("FAKE_OLD_VERSION", "1.0") // a jar in the station is a deployed one: the build raises its version
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.remove(f.src + "/gradle.properties")
	arrange(f)
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] gate ok\n")
	if _, after, ok := strings.Cut(f.calls(), "vm.sh <--src> <"+f.src+"/demo> <--expected-signer> <"); ok {
		signer, _, _ = strings.Cut(after, ">")
	}
	return signer, strings.TrimPrefix(f.err, warnNoBackup)
}

// The station's jars are read in addition to the home's, never instead of them (bench Δ24): when the
// station gives no signer to expect (only the development key, or two signers), the jars that were in
// <NIAGARA_HOME>/modules before the build still give theirs, as they did before the station was read.
func TestTheHomeStillGivesTheSignerWhenTheStationGivesNone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		station func(f *fixture)
	}{
		{"the station's jar carries the development signer", func(f *fixture) {
			f.zipJar(f.deployedJar("rt"), signedBy("NIAGARA4"))
		}},
		{"the station's jars carry two signers", func(f *fixture) {
			f.zipJar(f.deployedJar("rt"), signedBy("ACMESIGN"))
			f.zipJar(f.deployedJar("ux"), signedBy("OTHER"))
		}},
	} {
		signer, warning := signerGiven(t, func(f *fixture) {
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("ACMESIGN"))
			tc.station(f)
		})
		if signer != "ACMESIGN" || warning != "" {
			t.Errorf("%s: --expected-signer %q, stderr %q; want ACMESIGN and no warning", tc.name, signer, warning)
		}
	}
}

// A station jar whose signer cannot be told (it cannot be read, is not a zip, or holds a signature file
// without its block file) is not an unsigned one: the gate gets no signer, a warning names the jar and why,
// and the home's signer is not taken in its place.
func TestAStationJarWhoseSignerCannotBeToldGivesNoSignerAndAWarning(t *testing.T) {
	for _, tc := range []struct {
		name, why string
		station   func(f *fixture)
	}{
		{"a jar that cannot be read", "it cannot be read as a jar: open " + "{jar}" + ": permission denied", func(f *fixture) {
			f.zipJar(f.deployedJar("rt"), signedBy("ACMESIGN"))
			if err := os.Chmod(f.deployedJar("rt"), 0); err != nil {
				f.t.Fatal(err)
			}
			f.t.Cleanup(func() { os.Chmod(f.deployedJar("rt"), 0o644) })
		}},
		{"a jar that is not a zip", "it cannot be read as a jar: zip: not a valid zip file", func(f *fixture) {
			f.write(f.deployedJar("rt"), "garbage")
		}},
		{"a signature file without its block file", "META-INF/ACMESIGN.SF has no signature block file", func(f *fixture) {
			f.zipJar(f.deployedJar("rt"), map[string]string{"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\r\n\r\n", "META-INF/ACMESIGN.SF": "x"})
		}},
	} {
		if strings.Contains(tc.why, "permission denied") && os.Geteuid() == 0 {
			continue // root reads every file
		}
		var jar string
		signer, warning := signerGiven(t, func(f *fixture) {
			f.zipJar(withHome(f.t, f, true)+"/demo-rt.jar", signedBy("OTHER"))
			tc.station(f)
			jar = f.deployedJar("rt")
		})
		want := "[ng-deploy] WARN the signer of " + jar + " cannot be told (" + strings.ReplaceAll(tc.why, "{jar}", jar) + "), " +
			"so the gate accepts any signature and the jars of <NIAGARA_HOME>/modules are not asked in its place" + remedySaid
		if signer != "" || warning != want {
			t.Errorf("%s: --expected-signer %q, stderr %q, want none and %q", tc.name, signer, warning, want)
		}
	}
}
