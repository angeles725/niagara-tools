package ngdeploy

import (
	"strings"
	"testing"
)

// The move aside never replaces a file that is already aside: one that appears after the deploy looked (here
// during slotomatic, as another tool or a run without the lock could put it) stops the deploy before the
// build, and neither it nor the jar is touched.
func TestAMoveAsideNeverReplacesAFileAlreadyAside(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	planted := modules + "/demo-wb.jar" + asideSuffix
	f.write(f.dir+"/do/gradlew", `case "$*" in *slotomatic*) printf 'the only copy' > "`+planted+`" ;; esac`)
	code := f.run("--no-backup", "--with-slotomatic")
	want := "[ng-deploy] ERROR build not started: demo-wb.jar could not be moved aside in " + modules + " (File exists); nothing was replaced: " + modules + " holds the jars it held before the build\n"
	if code != ExitEnv || !strings.HasSuffix(f.err, want) || strings.Contains(f.calls(), ":jar") {
		t.Errorf("exit %d, stderr:\n%scalls:\n%s", code, f.err, f.calls())
	}
	if f.read(planted) != "the only copy" || f.read(modules+"/demo-wb.jar") != "home-old-wb" || f.read(modules+"/demo-rt.jar") != "home-old-rt" || f.exists(modules+"/demo-rt.jar"+asideSuffix) {
		t.Errorf("the directory holds:\n%s", tree(t, modules))
	}
}

// When a deploy stops, a jar that <NIAGARA_HOME>/modules did not have is removed only when it is the jar this
// run built: one with other bytes (another tool put it there, or a jar that was put back) is left where it
// is and named, and the exit code is the check's.
func TestAJarThisRunDidNotBuildIsNotRemovedFromTheHome(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	t.Setenv("FAKE_VM_EXIT", "1")
	f.write(f.dir+"/do/gradlew", installing(f.dir+"/home", `printf 'not of this run' > "`+modules+`/demo-ux.jar"`, "rt"))
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR verify-module gate failed; jars not fit to deploy (use --no-gate to override)\n" +
		"[ng-deploy] WARN demo-ux.jar in " + modules + " was not there before the build and is not the jar this run built (build/libs): it is left where it is\n" +
		strings.ReplaceAll(homeForeignLeft, "{modules}", modules)
	if code != ExitVerify || !strings.HasSuffix(f.err, want) {
		t.Errorf("exit %d, stderr:\n%s", code, f.err)
	}
	if f.read(modules+"/demo-ux.jar") != "not of this run" || f.read(modules+"/demo-rt.jar") != "home-old-rt" {
		t.Errorf("the directory holds:\n%s", tree(t, modules))
	}
}

const homeMovedBack = "[ng-deploy] NOTICE: the build installed no jar in {modules}; the module jars moved aside for it were moved back: it holds the module jars it held before the build\n"

// The NOTICE says the jars of the build were taken out only when the build installed any: a build that
// fails before it installs one moved nothing in, and the line says the jars aside were moved back.
func TestTheNoticeSaysWhatTheBuildDid(t *testing.T) {
	f := newFixture(t)
	modules := withHome(t, f, true)
	before := tree(t, modules)
	f.write(f.dir+"/do/gradlew", "exit 1")
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR build failed (gradlew exited non-zero)\n" + strings.ReplaceAll(homeMovedBack, "{modules}", modules)
	if code != ExitBuild || f.err != want || tree(t, modules) != before {
		t.Errorf("exit %d, stderr:\n%s", code, f.err)
	}
}

// A directory under the name of a jar of the mode in <NIAGARA_HOME>/modules is refused before the build in
// every mode, not only when the station is that directory: the put-back could not rename over it
// (retro 2026-11-15-coordination-stretch-deltas D8). Exit 40, nothing ran, nothing changed.
func TestADirectoryUnderAJarNameInTheHomeIsRefusedBeforeTheBuild(t *testing.T) {
	for _, args := range [][]string{{"--no-backup"}, {"--no-backup", "--no-deploy"}, {"--no-backup", "--mode", "C"}} {
		f := newFixture(t)
		modules := withHome(t, f, true)
		f.remove(modules + "/demo-rt.jar")
		f.write(modules+"/demo-rt.jar/keep", "kept")
		before := tree(t, modules)
		code := f.run(args...)
		want := "[ng-deploy] ERROR build refused for demo-rt.jar: " + modules + " holds a directory under that name, not a jar file; nothing was replaced: " + modules + " holds the jars it held before the build\n"
		if code != ExitCopy || f.err != want || f.calls() != "" || tree(t, modules) != before {
			t.Errorf("%v: exit %d, stderr:\n%scalls %q", args, code, f.err, f.calls())
		}
	}
}
