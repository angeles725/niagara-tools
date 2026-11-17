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

// The two settings name one directory however STATION_MODULES_DIR spells it: relative to the working
// directory, with "./", or through a symbolic link above it. A directory that only looks alike (a sibling
// home's modules) is another one, and is copied to.
func TestAStationSpelledAnotherWayIsStillTheHomeModulesDirectory(t *testing.T) {
	for name, spelling := range map[string]string{
		"relative":                  "../home/modules",
		"relative with ./":          "./../home/./modules/",
		"through a linked parent":   "{dir}/linked/modules",
		"relative through the link": "../linked/modules",
	} {
		f := newFixture(t)
		modules := sameDir(t, f)
		if err := os.Symlink(f.dir+"/home", f.dir+"/linked"); err != nil {
			t.Fatal(err)
		}
		f.set("STATION_MODULES_DIR", strings.ReplaceAll(spelling, "{dir}", f.dir))
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		f.want(f.run("--no-backup"), ExitOK, strings.ReplaceAll(sameDirSaid, "{modules}", modules), "[ng-deploy] copy: nothing to copy, the build installed demo-rt.jar demo-ux.jar in "+modules)
		if f.err != warnNoBackup || f.read(modules+"/demo-rt.jar") != "rtjar" || strings.Contains(tree(t, modules), ".ng-deploy") {
			t.Errorf("%s: stderr %q, the directory holds:\n%s", name, f.err, tree(t, modules))
		}
	}

	f := newFixture(t)
	modules := sameDir(t, f)
	f.write(f.dir+"/home2/modules/demo-rt.jar", "station-old-rt")
	f.set("STATION_MODULES_DIR", "../home2/modules")
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
	if strings.Contains(f.out, "nothing to copy") || strings.Contains(f.out, "is <NIAGARA_HOME>/modules") || f.read(f.dir+"/home2/modules/demo-rt.jar") != "rtjar" || f.read(modules+"/demo-rt.jar") != "rtjar" {
		t.Errorf("a directory that is not the home's: stdout:\n%s", f.out)
	}
}

// The same directory, and it holds something that is not a jar file under the name of a jar of the mode:
// the deploy refuses as the copy does for any station (exit 40), here before the build moves anything.
func TestAStationThatIsTheHomeModulesDirectoryRefusesADirectoryUnderAJarName(t *testing.T) {
	f := newFixture(t)
	modules := sameDir(t, f)
	f.remove(modules + "/demo-ux.jar")
	f.write(modules+"/demo-ux.jar/keep", "kept")
	before := tree(t, modules)
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR copy failed for demo-ux.jar: the station holds a directory under that name, not a jar file; nothing was replaced: the station holds the jars it held before the deploy\n"
	if code != ExitCopy || f.err != want || f.calls() != "" || tree(t, modules) != before {
		t.Errorf("exit %d, stderr:\n%scalls %q", code, f.err, f.calls())
	}
}

// --no-deploy builds without the prechecks, the gate or the copy, and keeps what gradle installed in
// <NIAGARA_HOME>/modules: when that directory is the station's, the station would get the unchecked jars.
// The run is refused before the backup and before gradle (exit 10), the directory left as it was; the same
// directory without --no-deploy still deploys through the checks (bench Δ23).
func TestNoDeployIntoTheStationThatIsTheHomeModulesDirectoryIsRefused(t *testing.T) {
	for _, viaEnv := range []bool{false, true} {
		f := newFixture(t)
		modules := sameDir(t, f)
		args := []string{"--no-backup", "--no-deploy"}
		if viaEnv {
			f.set("NO_DEPLOY", "1")
			args = args[:1]
		}
		before := tree(t, modules)
		code := f.run(args...)
		want := "[ng-deploy] ERROR --no-deploy refused: STATION_MODULES_DIR is <NIAGARA_HOME>/modules (" + modules +
			"): gradle's :jar step installs the built jars there and --no-deploy keeps them, so the station would get jars no check has passed; " +
			"deploy without --no-deploy (the copy step then verifies them), or point STATION_MODULES_DIR or NIAGARA_HOME elsewhere; nothing was built or changed\n"
		if code != ExitEnv || f.err != want || f.calls() != "" || tree(t, modules) != before {
			t.Errorf("env=%v: exit %d, stderr %q, calls %q, the directory holds:\n%s", viaEnv, code, f.err, f.calls(), tree(t, modules))
		}
	}
}
