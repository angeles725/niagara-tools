package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// sameDir makes STATION_MODULES_DIR another spelling (a symbolic link) of <NIAGARA_HOME>/modules, which
// holds an older rt and ux jar.
func sameDir(t *testing.T, f *fixture) (modules string) {
	t.Helper()
	modules = withHome(t, f, true)
	f.write(modules+"/demo-ux.jar", "home-old-ux")
	if err := os.MkdirAll(f.dir+"/via", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(modules, f.dir+"/via/modules"); err != nil {
		t.Fatal(err)
	}
	f.set("STATION_MODULES_DIR", f.dir+"/via/modules")
	return modules
}

const sameDirSaid = "[ng-deploy] STATION_MODULES_DIR is <NIAGARA_HOME>/modules ({modules}): the build installs the jars there, so the copy step only verifies them\n"

// When the station directory is <NIAGARA_HOME>/modules the deploy says so, the version precheck compares
// the build with the jar that was there before gradle replaced it, and the copy step copies nothing: it
// checks that the directory holds the built jars.
func TestAStationThatIsTheHomeModulesDirectoryIsVerifiedNotCopied(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run("--no-backup"), ExitOK, strings.ReplaceAll(sameDirSaid, "{modules}", modules),
		"[ng-deploy] version-bump ok (demo-rt.jar: 1.0 -> 1.1)\n[ng-deploy] version-bump ok (demo-ux.jar: 1.0 -> 1.1)\n",
		"[ng-deploy] gate ok\n[ng-deploy] copy: nothing to copy, the build installed demo-rt.jar demo-ux.jar in "+modules+" (the same bytes as build/libs)\n[ng-deploy] copy ok\n")
	if f.err != warnNoBackup || f.read(modules+"/demo-rt.jar") != "rtjar" || f.read(modules+"/demo-ux.jar") != "uxjar" || f.read(modules+"/demo-wb.jar") != "home-old-wb" {
		t.Errorf("stderr %q, the directory holds:\n%s", f.err, tree(t, modules))
	}
	if strings.Contains(tree(t, modules), ".ng-deploy-") || !f.exists(f.work+"/.last-deploy-sha") {
		t.Errorf("files of the deploy left, or no record:\n%s", tree(t, modules))
	}
}

// The same directory, and the built version is not higher than the one that was deployed: exit 54 as for
// any station, and the directory holds the previous jars again.
func TestAStationThatIsTheHomeModulesDirectoryStillGetsTheVersionPrecheck(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	t.Setenv("FAKE_OLD_VERSION", "2.0")
	t.Setenv("FAKE_NEW_VERSION", "1.0")
	before := tree(t, modules)
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck: demo-rt.jar vendorVersion went down (2.0 -> 1.0) — a deploy must raise it; bump defaultModuleVersion above 2.0 and rebuild\n" +
		strings.ReplaceAll(homeRestored, "{modules}", modules)
	if code != ExitVersionDown || f.err != want || tree(t, modules) != before {
		t.Errorf("exit %d, stderr:\n%sthe directory holds:\n%s", code, f.err, tree(t, modules))
	}
}

// The same directory, and the build did not leave the built jar there: nothing is copied over it; the
// deploy fails (exit 40) and the previous jars are put back.
func TestAStationThatIsTheHomeModulesDirectoryMustHoldTheBuiltJars(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", "", "rt")) // the ux jar is not installed
	before := tree(t, modules)
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR copy failed for demo-ux.jar: STATION_MODULES_DIR is <NIAGARA_HOME>/modules and the jar the build left there is missing or is not the one in build/libs\n" +
		strings.ReplaceAll(homeRestored, "{modules}", modules)
	if code != ExitCopy || f.err != want || tree(t, modules) != before || f.exists(f.work+"/.last-deploy-sha") {
		t.Errorf("exit %d, stderr:\n%sthe directory holds:\n%s", code, f.err, tree(t, modules))
	}
}
