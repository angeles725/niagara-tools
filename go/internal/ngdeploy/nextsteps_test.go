package ngdeploy

import (
	"strings"
	"testing"
)

// closing is what a deploy that went through prints after its last line of before: what it wrote and
// where, who signed it and who decides on trust, what to do next, and how to take the module out again.
func closing(f *fixture, mode, signers string) string {
	rt, ux := f.modules+"/demo-rt.jar", f.modules+"/demo-ux.jar"
	names, paths := "demo-rt.jar demo-ux.jar", rt+" "+ux
	switch mode {
	case "B":
		names, paths = "demo-ux.jar", ux
	case "C":
		names, paths = "demo-rt.jar", rt
	}
	out := "[ng-deploy] done: " + names + " copied to " + f.modules + "\n" +
		"[ng-deploy] signer: " + signers + "; the gate checks who signed a jar, not whether the station trusts that signer: the station decides when it loads the module\n"
	if mode != "B" {
		lab := f.dir + "/repo/scripts/../build-n4-module-kit/toolbelt/lab-station.sh"
		out += "[ng-deploy] next: restart the station; ng-deploy did not look at whether one is running, and stops or starts none\n" +
			"[ng-deploy] next: stop it with: " + lab + " stop --home <NIAGARA_HOME> <station>; start it with: " + lab + " start --home <NIAGARA_HOME> <station> (lab-station stops only a station it started; a Windows service or station.exe is stopped and started there)\n" +
			"[ng-deploy] next: after the restart, run the mcp-n4 prompt post-deploy-check with station=<station> module=demo (read-only: restart_needed, module version, health, new alarms)\n" +
			"[ng-deploy] next: after the restart, check the station console for the module: " + f.dir + "/repo/scripts/../build-n4-module-kit/toolbelt/triage-console.sh --package <the Java package of the module> --module-prefix demo <station-dir>/console*.txt (exit 1: rows to look at)\n"
	}
	return out + "[ng-deploy] undo: to remove the module again, take its components out of the station, stop the station and delete " + paths + "\n" + noTargetLine
}

// A deploy that went through ends with the lines it always ended with, then the closing lines, for the jars
// of its mode; the signer line names the signer of each jar written, or says a jar carries no signature.
func TestADeployThatWentThroughEndsWithWhatToDoNext(t *testing.T) {
	const unsigned = "demo-rt.jar carries no complete signature, demo-ux.jar carries no complete signature"
	for _, tc := range []struct{ mode, before, signers string }{
		{"A", "[ng-deploy] gate ok\n[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n", unsigned},
		{"B", "[ng-deploy] copy ok\n[ng-deploy] browser hard-reload only (no station restart needed)\n", "demo-ux.jar carries no complete signature"},
		{"C", "[ng-deploy] gate ok\n[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n", "demo-rt.jar carries no complete signature"},
	} {
		f := newFixture(t)
		code := f.run("--no-backup", "--mode", tc.mode)
		if want := tc.before + closing(f, tc.mode, tc.signers); code != ExitOK || !strings.HasSuffix(f.out, want) || f.err != warnNoBackup {
			t.Errorf("mode %s: exit %d, stdout:\n%swant the end:\n%s", tc.mode, code, f.out, want)
		}
	}
	f := newFixture(t)
	f.zipJar(f.builtJar("rt"), signedBy("ACMESIGN"))
	f.zipJar(f.builtJar("ux"), signedBy("ACMESIGN", "OTHER"))
	f.want(f.run("--no-backup"), ExitOK, closing(f, "A", "demo-rt.jar signed by ACMESIGN, demo-ux.jar signed by ACMESIGN and OTHER"))

	// The undo line points at the backup only when this run wrote one.
	f = newFixture(t)
	f.deployed()
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run(), ExitOK, " and delete "+f.modules+"/demo-rt.jar "+f.modules+"/demo-ux.jar (for the version before this one, restore the backup this run wrote: --help, Backups)\n")
	f = newFixture(t)
	f.want(f.run(), ExitOK, "nothing to archive\n", " and delete "+f.modules+"/demo-rt.jar "+f.modules+"/demo-ux.jar\n")
}

// Nothing of that is printed by a run that deployed nothing or did not finish: a build with --no-deploy,
// a check that failed, and a deploy whose commit could not be recorded (exit 41) end as they ended.
func TestTheClosingLinesArePrintedOnlyByADeployThatWentThrough(t *testing.T) {
	for name, tc := range map[string]struct {
		arrange func(f *fixture)
		args    []string
		code    int
		last    string
	}{
		"--no-deploy":       {func(*fixture) {}, []string{"--no-deploy"}, ExitOK, "[ng-deploy] no-deploy: jars in build/libs/\n"},
		"a check failed":    {func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, nil, ExitVerify, "[ng-deploy] gate: verify-module on demo-rt.jar demo-ux.jar\n"},
		"no record written": {func(f *fixture) { f.t.Setenv("FAKE_GIT_HEAD_EXIT", "128") }, nil, ExitRecord, "[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n"},
	} {
		f := newFixture(t)
		tc.arrange(f)
		code := f.run(append([]string{"--no-backup"}, tc.args...)...)
		if code != tc.code || !strings.HasSuffix(f.out, tc.last) || strings.Contains(f.out, "done:") || strings.Contains(f.out+f.err, "undo:") {
			t.Errorf("%s: exit %d, stdout:\n%s", name, code, f.out)
		}
	}
}
