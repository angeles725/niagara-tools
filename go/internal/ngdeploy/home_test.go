package ngdeploy

import (
	"os"
	"strings"
	"syscall"
	"testing"
)

// installing is a gradlew that does what the vendor's plugin does on :jar: it installs the built jars it
// is given (profiles) into <NIAGARA_HOME>/modules, then runs after.
func installing(home string, after string, profiles ...string) string {
	script := `case "$*" in *:jar*)` + "\n"
	for _, p := range profiles {
		script += `cp "${0%/*}/demo/demo-` + p + `/build/libs/demo-` + p + `.jar" "` + home + `/modules/demo-` + p + `.jar"` + "\n"
	}
	return script + after + "\n;; esac\n" + `exit "${FAKE_GRADLEW_EXIT:-0}"`
}

// withHome gives the fixture a niagara_home of its own, a mirror or not, whose modules directory holds an
// older rt jar (with a mode of its own) and an older wb jar, and a gradlew that installs the rt and ux jars
// there. It returns the modules directory.
func withHome(t *testing.T, f *fixture, mirror bool) (modules string) {
	t.Helper()
	home := f.dir + "/home"
	modules = home + "/modules"
	f.write(modules+"/demo-rt.jar", "home-old-rt")
	f.write(modules+"/demo-wb.jar", "home-old-wb")
	f.write(modules+"/other-rt.jar", "another module")
	if err := os.Chmod(modules+"/demo-rt.jar", 0o640); err != nil {
		t.Fatal(err)
	}
	if mirror {
		f.write(home+"/.niagara-mirror", "")
	}
	f.set("NIAGARA_HOME", home)
	f.write(f.dir+"/do/gradlew", installing(home, "", "rt", "ux"))
	return modules
}

const realHomeRefused = "[ng-deploy] ERROR NIAGARA_HOME {home} is a real niagara_home (no .niagara-mirror): gradle's :jar step would replace the jars of its modules/ before any check runs; " +
	"build against a mirror (toolbelt/mirror-niagara-home.sh <niagara_home> <mirror_dir>), or pass --install-into-home\n"

// A niagara_home that is there and is not a mirror is refused before the backup and before gradle runs,
// slotomatic included: nothing ran and nothing in it changed. The option build.sh has opts in, from the
// command line only.
func TestAHomeThatIsNotAMirrorIsRefusedBeforeGradleRuns(t *testing.T) {
	for _, args := range [][]string{nil, {"--with-slotomatic"}, {"--no-deploy"}} {
		f := newFixture(t)
		modules := withHome(t, f, false)
		before := tree(t, f.dir+"/home")
		code := f.run(args...)
		if want := strings.ReplaceAll(realHomeRefused, "{home}", f.dir+"/home"); code != ExitEnv || f.err != want || f.out != "" || f.calls() != "" {
			t.Errorf("%v: exit %d, stdout %q, stderr %q, calls %q", args, code, f.out, f.err, f.calls())
		}
		if after := tree(t, f.dir+"/home"); after != before || f.exists(f.work+"/_backups") || f.exists(modules+"/demo-ux.jar") {
			t.Errorf("%v: the refused run changed something:\n%s", args, after)
		}
	}

	f := newFixture(t)
	modules := withHome(t, f, false)
	f.want(f.run("--no-backup", "--install-into-home"), ExitOK, "[ng-deploy] copy ok\n")
	if f.read(modules+"/demo-rt.jar") != "rtjar" || f.read(f.deployedJar("rt")) != "rtjar" {
		t.Errorf("with --install-into-home the deploy did not go through: stderr %q", f.err)
	}

	f = newFixture(t)
	withHome(t, f, false)
	f.set("INSTALL_INTO_HOME", "1")
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR env file " + f.env + ": line 10: INSTALL_INTO_HOME would switch the refusal of a real niagara_home off, which an env file may not do: give --install-into-home on the command line\n"
	if code != ExitEnv || f.err != want || f.calls() != "" {
		t.Errorf("INSTALL_INTO_HOME=1 in the env file: exit %d, stderr %q", code, f.err)
	}
}

// What a deploy printed and did before the guard still holds for a mirror and for a NIAGARA_HOME that
// names no directory (gradle fails on that one by itself): no new line, the same calls.
func TestAMirrorHomeAndAHomeThatIsNotThereDeployAsBefore(t *testing.T) {
	f := newFixture(t) // the fixture's NIAGARA_HOME names no directory
	f.want(f.run("--no-backup"), ExitOK)
	out, calls := f.out, strings.ReplaceAll(strings.ReplaceAll(f.calls(), "/n4/niagara home", "HOME"), f.dir, "DIR")
	if f.err != warnNoBackup {
		t.Errorf("stderr %q", f.err)
	}
	f = newFixture(t)
	modules := withHome(t, f, true)
	f.want(f.run("--no-backup"), ExitOK)
	if f.out != out || f.err != warnNoBackup || strings.ReplaceAll(strings.ReplaceAll(f.calls(), f.dir+"/home", "HOME"), f.dir, "DIR") != calls {
		t.Errorf("a mirror changed the run:\n%s%s%s", f.out, f.err, f.calls())
	}
	// The build's jars stay in the mirror, the wb jar it did not build is as it was, nothing is left aside.
	if f.read(modules+"/demo-rt.jar") != "rtjar" || f.read(modules+"/demo-ux.jar") != "uxjar" || f.read(modules+"/demo-wb.jar") != "home-old-wb" || f.read(modules+"/other-rt.jar") != "another module" {
		t.Errorf("the mirror holds:\n%s", tree(t, modules))
	}
	if left := strings.Count(tree(t, modules), ".ng-deploy-"); left != 0 {
		t.Errorf("files of the deploy left in the mirror:\n%s", tree(t, modules))
	}
}

const homeRestored = "[ng-deploy] NOTICE: the jars the build installed in {modules} were taken out again: it holds the module jars it held before the build\n"

// Whatever stops the deploy after gradle installed the built jars in <NIAGARA_HOME>/modules, that directory
// ends as it was before the build: the previous jar is back, bytes and mode, the jar it did not have is
// gone, and the exit code and the ERROR line are the check's own.
func TestACheckThatFailsPutsTheHomeJarsBack(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		code    int
		errLine string
	}{
		{"type count", func(f *fixture) { f.t.Setenv("FAKE_UNZIP_TYPES", "3") }, ExitVerify, "verify failed: expected 9 <type entries in "},
		{"gate", func(f *fixture) { f.t.Setenv("FAKE_VM_EXIT", "1") }, ExitVerify, "verify-module gate failed"},
		{"version went down", func(f *fixture) {
			f.deployed()
			f.t.Setenv("FAKE_OLD_VERSION", "2.0")
			f.t.Setenv("FAKE_NEW_VERSION", "1.0")
		}, ExitVersionDown, "version-bump precheck: demo-rt.jar vendorVersion went down"},
		{"schema risk", func(f *fixture) { f.baseline(); f.t.Setenv("FAKE_SR_EXIT", "2") }, ExitSchema, "schema-risk verdict OUTAGE"},
		{"the build fails after it installed a jar", func(f *fixture) {
			f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", "rm -f '"+f.dir+"/home/modules/demo-rt.jar'; cp '"+f.builtJar("ux")+"' '"+f.dir+"/home/modules/demo-ux.jar'; exit 1"))
		}, ExitBuild, "build failed (gradlew exited non-zero)"},
		{"the copy fails", func(f *fixture) { failRename(f.t, renameInUX) }, ExitCopy, "copy failed for demo-ux.jar: Permission denied; "},
	} {
		f := newFixture(t)
		modules := withHome(t, f, true)
		tc.arrange(f)
		before := tree(t, f.dir+"/home")
		code := f.run("--no-backup")
		notice := strings.ReplaceAll(homeRestored, "{modules}", modules)
		if code != tc.code || !strings.Contains(f.err, "[ng-deploy] ERROR "+tc.errLine) || !strings.HasSuffix(f.err, notice) || strings.Count(f.err, "ERROR") != 1 {
			t.Errorf("%s: exit %d, stderr:\n%s", tc.name, code, f.err)
		}
		if after := tree(t, f.dir+"/home"); after != before {
			t.Errorf("%s: <NIAGARA_HOME> is not as before the build:\n%swas:\n%s", tc.name, after, before)
		}
	}
}

// A previous jar that cannot be moved back, and an added jar that cannot be removed, are named jar by jar
// in the form the copy uses; the exit code stays the check's.
func TestAHomeThatCannotBePutBackIsReportedJarByJar(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	t.Setenv("FAKE_VM_EXIT", "1")
	failRename(t, "/home/modules/demo-rt.jar"+asideSuffix+">/home/modules/demo-rt.jar")
	failRemove(t, "/home/modules/demo-ux.jar")
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR " + modules + " does NOT hold the jars it held before the build: " +
		"demo-rt.jar is the NEW jar and the previous one is in demo-rt.jar.ng-deploy-old (Permission denied), " +
		"demo-ux.jar, which " + modules + " did not have, could not be removed (Device or resource busy): remove it before the next build or deploy, " +
		"demo-wb.jar is the previous jar; put each named .ng-deploy-old file back under its jar name before the next build or deploy\n"
	if code != ExitVerify || !strings.HasSuffix(f.err, "[ng-deploy] ERROR verify-module gate failed; jars not fit to deploy (use --no-gate to override)\n"+want) {
		t.Errorf("exit %d, stderr:\n%s", code, f.err)
	}
	if f.read(modules+"/demo-rt.jar"+asideSuffix) != "home-old-rt" || f.read(modules+"/demo-wb.jar") != "home-old-wb" {
		t.Errorf("the previous jars are not where the line says:\n%s", tree(t, modules))
	}
}

// A jar left aside in <NIAGARA_HOME>/modules by a run that was killed is never removed: the deploy refuses
// before the backup and before gradle, with the lines the copy uses for the station.
func TestAJarLeftAsideInTheHomeIsRefusedBeforeGradleRuns(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	f.write(modules+"/demo-wb.jar"+asideSuffix, "prev-wb")
	f.write(modules+"/demo-ux.jar"+asideSuffix, "prev-ux")
	before := tree(t, modules)
	code := f.run("--with-slotomatic")
	want := "[ng-deploy] ERROR build refused: a deploy that did not finish left previous jars aside in " + modules + "; nothing was built or changed\n" +
		strings.ReplaceAll(asideNoJar, "JAR", "demo-ux.jar") + strings.ReplaceAll(asideWithJar, "JAR", "demo-wb.jar") + asideWaysOut
	if code != ExitEnv || f.err != want || f.calls() != "" || tree(t, modules) != before || f.exists(f.work+"/_backups") {
		t.Errorf("exit %d, stderr:\n%scalls %q", code, f.err, f.calls())
	}
}

// A jar of the home that cannot be moved aside stops the deploy before gradle, with the others moved back.
func TestAHomeJarThatCannotBeMovedAsideStopsTheDeployBeforeTheBuild(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	failRename(t, "/home/modules/demo-wb.jar"+asideSuffix)
	before := tree(t, modules)
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR build not started: demo-wb.jar could not be moved aside in " + modules + " (Permission denied); nothing was replaced: " + modules + " holds the jars it held before the build\n"
	if code != ExitEnv || f.err != want || strings.Contains(f.calls(), "gradlew") || tree(t, modules) != before {
		t.Errorf("exit %d, stderr:\n%scalls:\n%s", code, f.err, f.calls())
	}
}

// A SIGINT or SIGTERM once the home's jars are aside ends the run with 128 + the signal after they are put
// back: before the build (the signal arrives while a jar is moved aside) and during it (gradle installed).
func TestASignalBeforeTheCopyPutsTheHomeJarsBack(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	before := tree(t, modules)
	signalDuring(t, syscall.SIGTERM, "/home/modules/demo-rt.jar"+asideSuffix)
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR interrupted (SIGTERM) before the copy; nothing was copied to the station\n" + strings.ReplaceAll(homeRestored, "{modules}", modules)
	if code != 128+int(syscall.SIGTERM) || f.err != want || strings.Contains(f.calls(), "gradlew") || tree(t, modules) != before {
		t.Errorf("before the build: exit %d, stderr:\n%scalls:\n%s", code, f.err, f.calls())
	}

	f = newFixture(t)
	modules = withHome(t, f, true)
	before = tree(t, modules)
	signalDuring(t, syscall.SIGINT, "/no/rename/matches/this")
	f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", "kill -INT $PPID; command -p sleep 0.5; exit 130", "rt", "ux"))
	code = f.run("--no-backup")
	want = warnNoBackup + "[ng-deploy] ERROR build failed (gradlew exited non-zero)\n[ng-deploy] ERROR interrupted (SIGINT) before the copy; nothing was copied to the station\n" + strings.ReplaceAll(homeRestored, "{modules}", modules)
	if code != 128+int(syscall.SIGINT) || f.err != want || f.exists(f.deployedJar("rt")) || tree(t, modules) != before {
		t.Errorf("during the build: exit %d, stderr:\n%s", code, f.err)
	}
}

// A deploy that went through and cannot tidy <NIAGARA_HOME>/modules is still done (exit 0): the previous jar
// it could not remove, and the jar gradle did not install again that it could not move back, are warnings.
func TestADeployThatCannotTidyTheHomeWarnsAndIsDone(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	failRemove(t, "/home/modules/demo-rt.jar"+asideSuffix)
	failRename(t, "/home/modules/demo-wb.jar"+asideSuffix+">/home/modules/demo-wb.jar")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
	want := warnNoBackup + "[ng-deploy] WARN the previous jar demo-rt.jar.ng-deploy-old in " + modules + " could not be removed (Device or resource busy): delete it, the next deploy refuses while it is there\n" +
		"[ng-deploy] WARN demo-wb.jar in " + modules + " could not be moved back from demo-wb.jar.ng-deploy-old (Permission denied): rename it back, the next deploy refuses while it is there\n"
	if f.err != want || f.read(modules+"/demo-rt.jar") != "rtjar" || f.read(modules+"/demo-wb.jar"+asideSuffix) != "home-old-wb" {
		t.Errorf("stderr:\n%s", f.err)
	}
}
