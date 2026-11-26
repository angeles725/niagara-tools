package ngdeploy

import (
	"strings"
	"testing"
)

// ownState writes the module's own BUILD-STATE.md in its module root (the directory of its gradlew).
func (f *fixture) ownState(text string) string {
	f.t.Helper()
	path := f.src + "/BUILD-STATE.md"
	f.write(path, text)
	return path
}

// Rule: a station of another major.minor than the recorded niagara_target is exit 55 before anything runs
// (no gradle, no gate, no backup, no copy), naming both versions and the file.
func TestAStationOfAnotherNiagaraVersionThanTheRecordIsRefused(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	root := installed(f, "4.14.0.162")
	path := f.ownState(envelope("demo", "niagara_target: 4.15.6.20"))
	want := "[ng-deploy] ERROR niagara_target precheck: demo is recorded for Niagara 4.15.6.20 (niagara_target in " + path +
		") but the target installation " + root + " is Niagara 4.14.0.162; nothing was built or copied: " +
		"deploy into an installation of 4.15, or, if the station moved, update niagara_target there\n"
	code := f.run()
	if code != ExitTargetVersion || f.out != "" || f.err != want || f.calls() != "" || f.exists(f.work+"/_backups") {
		t.Errorf("exit %d, stdout %q, stderr %q, calls %q; want exit %d and stderr %q", code, f.out, f.err, f.calls(), ExitTargetVersion, want)
	}
	if got := f.read(f.deployedJar("rt")); got != "old-rt" {
		t.Errorf("the station's rt jar is now %q, want it unchanged", got)
	}
}

// Rule: a station of the recorded major.minor deploys, and one line says the record agrees; the kit's file is
// read when the module root has none, and the module's own file wins over it.
func TestAStationOfTheRecordedVersionDeploys(t *testing.T) {
	f := newFixture(t)
	root := installed(f, "4.15.6.20")
	compiledFor(f, "4.15")
	f.kitState(envelope("demo", "niagara_target: 4.14"))
	path := f.ownState(envelope("demo", "niagara_target: 4.15"))
	code := f.run("--no-backup")
	f.want(code, ExitOK, "[ng-deploy] niagara_target ok (4.15 of demo in "+path+"; "+root+" is Niagara 4.15.6.20)\n", "[ng-deploy] copy ok\n")

	g := newFixture(t)
	g.deployed()
	installed(g, "4.15.6.20")
	kit := g.kitState(envelope("demo", "niagara_target: 4.14"))
	g.want(g.run(), ExitTargetVersion, "err:niagara_target in "+kit+")")

	h := newFixture(t) // a module file with no envelope of the module records nothing
	installed(h, "4.15.6.20")
	compiledFor(h, "4.15")
	h.kitState(envelope("demo", "niagara_target: 4.14"))
	h.ownState(envelope("other", "niagara_target: 4.14"))
	h.want(h.run("--no-backup"), ExitOK)
}

// Rule: no record, `unknown` and --no-deploy change nothing; a station whose version cannot be read is a WARN and
// the deploy goes on to the jar-vs-home check.
func TestWithoutARecordOrAReadableStationTheDeployIsUnchanged(t *testing.T) {
	for _, state := range []string{"", envelope("demo"), envelope("demo", "niagara_target: unknown")} {
		f := newFixture(t)
		installed(f, "4.15.6.20")
		compiledFor(f, "4.15")
		if state != "" {
			f.ownState(state)
		}
		f.want(f.run("--no-backup"), ExitOK)
		if strings.Contains(f.out, "niagara_target") {
			t.Errorf("stdout mentions the record: %q", f.out)
		}
	}
	f := newFixture(t) // the station holds no baja.jar
	path := f.ownState(envelope("demo", "niagara_target: 4.15.6.20"))
	f.want(f.run("--no-backup"), ExitOK, "err:[ng-deploy] WARN niagara_target 4.15.6.20 ("+path+") is not compared with "+f.modules+": ")
}

// Rule: a malformed niagara_target is exit 10 before anything runs, with the sweep's text and the file.
func TestAMalformedTargetRecordIsRefusedBeforeTheCopy(t *testing.T) {
	f := newFixture(t)
	path := f.ownState(envelope("demo", "niagara_target: 4.x"))
	f.wantRefused(f.run(), "[ng-deploy] ERROR envelope (demo) niagara_target invalid: '4.x' (want "+
		"2 to 4 dotted numbers such as 4.15 or 4.15.6.20, or unknown) in "+path+"\n")
}

// Rule: expected_signer is read from the module's own file too.
func TestTheExpectedSignerIsReadFromTheModuleRootFile(t *testing.T) {
	f := newFixture(t)
	f.kitState(envelope("demo", "expected_signer: kitsigner"))
	path := f.ownState(envelope("demo", "expected_signer: acmesigner2"))
	f.deployed()
	want := "[ng-deploy] ERROR niagaraSigningAlias=acmesigner is configured but the build-state envelope of demo " +
		"records expected_signer: acmesigner2 (" + path + ").\n" +
		"  Deploy with that alias (--gradle-prop niagaraSigningAlias=acmesigner2) or, if the signer changed, " +
		"update expected_signer there.\n"
	if code := f.run(); code != ExitEnv || f.err != want {
		t.Errorf("exit %d, stderr %q; want exit 10 and %q", code, f.err, want)
	}
}
