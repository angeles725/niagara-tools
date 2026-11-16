package ngdeploy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// tree is every entry under dir: its kind and mode, its path, and its content or link target.
func tree(t *testing.T, dir string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, _ := e.Info()
		b.WriteString(info.Mode().String() + " " + path)
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			to, _ := os.Readlink(path)
			b.WriteString(" -> " + to)
		case info.Mode().IsRegular():
			text, _ := os.ReadFile(path)
			b.WriteString(" [" + string(text) + "]")
		}
		b.WriteString("\n")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

const (
	asideRefused = "[ng-deploy] ERROR copy refused: a deploy that did not finish left previous jars aside in MODULES; nothing was copied or changed\n"
	asideWaysOut = "[ng-deploy]   to put the previous jars back, rename each .ng-deploy-old file over its jar; to keep what is there, delete the .ng-deploy-old files; then deploy again\n"
	asideWithJar = "[ng-deploy]   JAR present, JAR.ng-deploy-old present: the jar may be the new one of that deploy; the .ng-deploy-old file is the previous jar\n"
	asideNoJar   = "[ng-deploy]   JAR missing, JAR.ng-deploy-old present: the .ng-deploy-old file is the previous jar\n"
)

// A jar aside that this run did not put there may be the only copy of the previous module, beside its jar
// or without it: the deploy deletes none, changes nothing in the station, and refuses before it copies
// anything, saying what the station holds for each such jar and the two ways out. It looks at every jar
// the tool can deploy, not only the jars of the mode.
func TestADeployNeverDeletesAJarLeftAside(t *testing.T) {
	rt, ux := "demo-rt.jar", "demo-ux.jar"
	with := func(jar string) string { return strings.ReplaceAll(asideWithJar, "JAR", jar) }
	without := func(jar string) string { return strings.ReplaceAll(asideNoJar, "JAR", jar) }
	for _, tc := range []struct {
		name  string
		files map[string]string // the station, by file name
		args  []string
		want  string
	}{
		{"killed between the two renames in", map[string]string{rt: "NEW-rt", rt + asideSuffix: "prev-rt", ux + asideSuffix: "prev-ux", ux + stagedSuffix: "NEW-ux"},
			nil, with(rt) + without(ux)},
		{"both aside beside their jars", map[string]string{rt: "NEW-rt", rt + asideSuffix: "prev-rt", ux: "NEW-ux", ux + asideSuffix: "prev-ux"}, nil, with(rt) + with(ux)},
		{"one aside beside its jar", map[string]string{rt: "old-rt", ux: "NEW-ux", ux + asideSuffix: "prev-ux"}, nil, with(ux)},
		{"one aside without its jar", map[string]string{rt: "old-rt", ux + asideSuffix: "prev-ux"}, nil, without(ux)},
		{"aside for a jar outside the mode", map[string]string{rt: "old-rt", ux + asideSuffix: "prev-ux"}, []string{"--mode", "C"}, without(ux)},
	} {
		f := newFixture(t)
		for name, text := range tc.files {
			f.write(f.modules+"/"+name, text)
		}
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		before := tree(t, f.modules)
		code := f.run(append([]string{"--no-backup"}, tc.args...)...)
		want := warnNoBackup + strings.ReplaceAll(asideRefused, "MODULES", f.modules) + tc.want + asideWaysOut
		if code != ExitCopy || f.err != want || strings.Contains(f.out, "copy ok") || f.exists(f.work+"/.last-deploy-sha") {
			t.Errorf("%s: exit %d, stderr:\n%s", tc.name, code, f.err)
		}
		if after := tree(t, f.modules); after != before {
			t.Errorf("%s: the refusing run changed the station:\n%swas:\n%s", tc.name, after, before)
		}
	}
}

// failRemove makes the removal of a path ending in suffix fail.
func failRemove(t *testing.T, suffix string) {
	t.Helper()
	t.Cleanup(func() { remove = os.Remove })
	remove = func(path string) error {
		if strings.HasSuffix(path, suffix) {
			return &os.PathError{Op: "remove", Path: path, Err: syscall.EBUSY}
		}
		return os.Remove(path)
	}
}

// The copy went through but a jar aside cannot be removed: the deploy is done (exit 0), and the warning
// names the file and says the next deploy refuses until it is deleted.
func TestAJarAsideThatCannotBeRemovedAfterTheCopyIsNamed(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRemove(t, "/demo-ux.jar"+asideSuffix)
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n")
	want := warnNoBackup + "[ng-deploy] WARN the jars are copied, but the previous jar demo-ux.jar.ng-deploy-old could not be removed (Device or resource busy): delete it, the next deploy refuses while it is in the station\n"
	if f.err != want || f.read(f.deployedJar("ux")) != "uxjar" || f.read(f.deployedJar("ux")+asideSuffix) != "old-ux" || f.exists(f.deployedJar("rt")+asideSuffix) {
		t.Errorf("stderr %q", f.err)
	}
}

// Something that is not a jar file under the name of a jar (a directory, a link to nothing) is refused
// before anything is staged or moved, as the copy with cp refused it: the deploy never moves or removes it.
func TestATargetThatIsNotAJarFileIsRefusedBeforeAnythingIsStaged(t *testing.T) {
	for what, arrange := range map[string]func(f *fixture){
		"a directory":                func(f *fixture) { f.write(f.deployedJar("rt")+"/keep/inside", "precious") },
		"a symbolic link to nothing": func(f *fixture) { os.Symlink("/nonexistent/x.jar", f.deployedJar("rt")) },
		"a symbolic link to a directory": func(f *fixture) {
			f.write(f.dir+"/elsewhere/inside", "precious")
			os.Symlink(f.dir+"/elsewhere", f.deployedJar("rt"))
		},
	} {
		f := newFixture(t)
		f.write(f.deployedJar("ux"), "uxjar")
		arrange(f)
		before := tree(t, f.modules)
		code := f.run("--no-backup")
		want := warnNoBackup + "[ng-deploy] ERROR copy failed for demo-rt.jar: the station holds " + what + " under that name, not a jar file; " + copyUnchanged
		if code != ExitCopy || f.err != want {
			t.Errorf("%s: exit %d, stderr %q", what, code, f.err)
		}
		if after := tree(t, f.modules); after != before || f.read(f.dir+"/elsewhere/inside") != map[bool]string{true: "precious"}[strings.HasSuffix(what, "to a directory")] {
			t.Errorf("%s: the station changed:\n%swas:\n%s", what, after, before)
		}
	}
}

// A jar that is a symbolic link to a jar file is deployed, as before the staged copy: the link is replaced.
func TestATargetThatIsALinkToAJarFileIsReplaced(t *testing.T) {
	f := newFixture(t)
	f.write(f.dir+"/elsewhere/real-rt.jar", "rtjar") // the same bytes: no version to compare
	if err := os.Symlink(f.dir+"/elsewhere/real-rt.jar", f.deployedJar("rt")); err != nil {
		t.Fatal(err)
	}
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
	f.station("rtjar", "uxjar", 0o644)
}

// A new jar the station did not have and that cannot be removed again is the stuck item: the line says to
// remove it, and gives no instruction about files aside, since none is named.
func TestANewJarThatCannotBeRemovedAgainIsNamedWithoutTheAsideInstruction(t *testing.T) {
	f := newFixture(t)
	failRename(t, renameInUX)
	failRemove(t, "/demo-rt.jar")
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyMixed +
		"demo-rt.jar, which the station did not have, could not be removed (Device or resource busy): remove it before the station restarts\n"
	if code != ExitCopy || f.err != want || f.read(f.deployedJar("rt")) != "rtjar" {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}
