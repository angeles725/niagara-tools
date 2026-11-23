package ngdeploy

import (
	"strings"
	"testing"
)

// wbFixture is a mode D deploy: a built -wb jar, its type count, and the jars of a first deploy.
func wbFixture(t *testing.T) *fixture {
	t.Helper()
	f := newFixture(t)
	f.write(f.builtJar("wb"), "wbjar")
	f.set("EXPECTED_WB_TYPES", "9")
	t.Setenv("FAKE_OLD_VERSION", "1.0") // the version-bump precheck of a jar the station holds: 1.0 -> 1.1
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	return f
}

// Mode D gates the -rt and the -wb jar with verify-module (never a -ux jar) and backs up the jars the station
// holds, the -wb one included.
func TestModeDGatesAndBacksUpTheWbJar(t *testing.T) {
	f := wbFixture(t)
	f.write(f.deployedJar("rt"), "old-rt")
	f.write(f.deployedJar("wb"), "old-wb")
	f.want(f.run("--mode", "D"), ExitOK, "[ng-deploy] gate: verify-module on demo-rt.jar demo-wb.jar\n[ng-deploy] gate ok\n",
		"[ng-deploy] backup ok: _backups/demo-pre-")
	if want := "vm.sh <--src> <" + f.src + "/demo> <--expected-signer> <acmesigner> <" + f.builtJar("rt") + "> <" + f.builtJar("wb") + ">\n"; !strings.Contains(f.calls(), want) {
		t.Errorf("the gate did not run on the -rt and the -wb jar; calls:\n%s", f.calls())
	}
	if want := "tar <-czf> <_backups/demo-pre-TS.tar.gz> <-C> <" + f.modules + "> <demo-rt.jar> <demo-wb.jar>\n"; !strings.Contains(stamp(f.calls()), want) {
		t.Errorf("the backup did not archive the -rt and the -wb jar; calls:\n%s", f.calls())
	}
}

// A gate that fails on mode D is exit 51 and the station keeps the -wb jar it held.
func TestModeDGateFailureLeavesTheStationAlone(t *testing.T) {
	f := wbFixture(t)
	f.write(f.deployedJar("wb"), "old-wb")
	t.Setenv("FAKE_VM_EXIT", "1")
	f.want(f.run("--no-backup", "--mode", "D"), ExitVerify, "err:verify-module gate failed")
	if f.read(f.deployedJar("wb")) != "old-wb" || f.exists(f.deployedJar("rt")) || strings.Contains(f.out, "copy ok") {
		t.Errorf("a jar reached the station:\n%s", tree(t, f.modules))
	}
}

// A .ng-deploy-old of the -wb jar, left by an earlier mode D deploy that did not finish, stops the next mode D
// deploy with exit 40 before anything is copied; the file is the only copy of the previous jar and stays as it was.
func TestModeDRefusesALeftOverWbJarAside(t *testing.T) {
	f := wbFixture(t)
	f.write(f.deployedJar("wb")+".ng-deploy-old", "previous wb")
	before := tree(t, f.modules)
	code := f.run("--no-backup", "--mode", "D")
	want := warnNoBackup + strings.ReplaceAll(asideRefused, "MODULES", f.modules) +
		strings.NewReplacer("JAR.ng-deploy-old", "demo-wb.jar.ng-deploy-old", "JAR", "demo-wb.jar").Replace(asideNoJar) + asideWaysOut
	if code != ExitCopy || f.err != want || strings.Contains(f.out, "copy ok") {
		t.Errorf("exit %d, stderr:\n%s\nwant:\n%s", code, f.err, want)
	}
	if after := tree(t, f.modules); after != before || f.read(f.deployedJar("wb")+".ng-deploy-old") != "previous wb" || f.exists(f.deployedJar("rt")) {
		t.Errorf("the station changed:\nbefore:\n%s\nafter:\n%s", before, tree(t, f.modules))
	}
}

// A -wb jar that does not hold EXPECTED_WB_TYPES types is exit 50 after the -rt jar verified, with nothing copied.
func TestModeDWrongWbTypeCountExits50AndCopiesNothing(t *testing.T) {
	f := wbFixture(t)
	f.set("EXPECTED_WB_TYPES", "2")
	f.write(f.deployedJar("wb"), "old-wb")
	code := f.run("--no-backup", "--mode", "D")
	f.want(code, ExitVerify, "[ng-deploy] verify ok (demo-rt.jar: 9/9)\n", "err:expected 2 <type entries in "+f.builtJar("wb")+", found 9\n")
	if f.exists(f.deployedJar("rt")) || f.read(f.deployedJar("wb")) != "old-wb" || strings.Contains(f.calls(), "vm.sh") || strings.Contains(f.out, "copy ok") {
		t.Errorf("something was copied or gated after the mismatch:\n%s\ncalls:\n%s", tree(t, f.modules), f.calls())
	}
}
