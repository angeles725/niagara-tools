package ngdeploy

import (
	"strings"
	"testing"
)

func TestATypeCountMismatchExits50AndCopiesNothing(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "8")
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR verify failed: expected 9 <type entries in " + f.builtJar("rt") + ", found 8\n"
	if code != ExitVerify || f.err != want || f.exists(f.deployedJar("rt")) || f.exists(f.deployedJar("ux")) || strings.Contains(f.calls(), "vm.sh") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
	f = newFixture(t)
	f.set("EXPECTED_UX_TYPES", "2")
	f.want(f.run("--no-backup"), ExitVerify, "[ng-deploy] verify ok (demo-rt.jar: 9/9)\n", "err:expected 2 <type entries in "+f.builtJar("ux")+", found 9\n")
	if f.exists(f.deployedJar("rt")) {
		t.Error("the rt jar was copied before the ux jar was verified")
	}
}

// The stub prints a <types> wrapper line around the entries: only "<type " lines count.
func TestTheTypeCountIsTheEntriesNotTheWrapper(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] verify ok (demo-rt.jar: 9/9)\n")
	if !strings.Contains(f.calls(), "unzip <-p> <"+f.builtJar("rt")+"> <META-INF/module.xml>\n") {
		t.Errorf("calls:\n%s", f.calls())
	}
}

// A jar that declares no type is counted like any other: 0 of 0 passes, 0 of 9 is the mismatch of exit 50.
func TestAJarWithNoTypeEntryIsCountedAsZero(t *testing.T) {
	f := newFixture(t)
	f.set("EXPECTED_UX_TYPES", "0")
	t.Setenv("FAKE_UNZIP_TYPES", "0")
	f.want(f.run("--no-backup", "--mode", "B"), ExitOK, "[ng-deploy] verify ok (demo-ux.jar: 0/0)\n[ng-deploy] copy ok\n")
	if f.err != warnNoBackup || f.read(f.deployedJar("ux")) != "uxjar" {
		t.Errorf("0 of 0: stderr %q, deployed ux jar %q", f.err, f.read(f.deployedJar("ux")))
	}
	f = newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "0")
	code := f.run("--no-backup", "--mode", "B")
	want := warnNoBackup + "[ng-deploy] ERROR verify failed: expected 9 <type entries in " + f.builtJar("ux") + ", found 0\n"
	if code != ExitVerify || f.err != want || strings.Contains(f.out, "verify ok") || f.exists(f.deployedJar("ux")) {
		t.Errorf("0 of 9: exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
}

// An unzip that fails did not show the whole module.xml, whatever it printed: exit 50, never unzip's status.
func TestAModuleXMLThatCannotBeReadExits50(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		before  string // what the failed command wrote first
		status  string
	}{
		{"unzip fails after listing the entries", func(f *fixture) {
			f.t.Setenv("FAKE_UNZIP_EXIT", "9")
			f.t.Setenv("FAKE_UNZIP_ERR", "cannot find zipfile")
		}, "unzip: cannot find zipfile\n", "9"},
		{"unzip fails with nothing printed", func(f *fixture) { f.write(f.dir+"/do/unzip", "exit 3\n") }, "", "3"},
		{"exit 1 is not passed on as a bare fault", func(f *fixture) { f.write(f.dir+"/do/unzip", "exit 1\n") }, "", "1"},
		{"no unzip", func(f *fixture) { f.remove(f.bin + "/unzip") }, "ng-deploy.sh: unzip: command not found\n", "127"},
	} {
		f := newFixture(t)
		tc.arrange(f)
		code := f.run("--no-backup", "--mode", "C")
		want := warnNoBackup + tc.before + "[ng-deploy] ERROR verify failed: cannot read META-INF/module.xml in " + f.builtJar("rt") + " (unzip exit " + tc.status + ")\n"
		if code != ExitVerify || f.err != want || strings.Contains(f.out, "verify ok") || f.exists(f.deployedJar("rt")) || strings.Contains(f.calls(), "vm.sh") {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", tc.name, code, f.out, f.err)
		}
	}
}

func TestCacheBusterIsCheckedInTheUxJarWhenBuildIDIsSet(t *testing.T) {
	for _, tc := range []struct {
		name, id, index, mode string
		code                  int
		want                  string
	}{
		{"found", "b42", `<script src="app.js?v=b42"></script>`, "B", ExitOK, "[ng-deploy] verify ok (cache-buster ?v=b42 found)\n"},
		{"found in mode A", "b42", "x?v=b42", "A", ExitOK, "[ng-deploy] verify ok (cache-buster ?v=b42 found)\n"},
		{"another build", "b42", `<script src="app.js?v=b41"></script>`, "A", ExitVerify, "err:[ng-deploy] ERROR verify failed: index.html in JAR does not contain ?v=b42 (BUILD_ID check)\n"},
		{"the id is text, not a pattern", "a*b", "x?v=aXXb", "B", ExitVerify, "err:does not contain ?v=a*b (BUILD_ID check)\n"},
		{"mode C has no ux jar", "b42", "", "C", ExitOK, "[ng-deploy] copy ok\n"},
		{"no BUILD_ID, no check", "", "", "B", ExitOK, "[ng-deploy] copy ok\n"},
	} {
		f := newFixture(t)
		f.set("BUILD_ID", "'"+tc.id+"'")
		t.Setenv("FAKE_INDEX", tc.index)
		code := f.run("--no-backup", "--mode", tc.mode)
		f.want(code, tc.code, strings.ReplaceAll(tc.want, "JAR", f.builtJar("ux")))
		checked := strings.Contains(f.calls(), "unzip <-p> <"+f.builtJar("ux")+"> <rc/index.html>\n")
		if checked != (tc.id != "" && tc.mode != "C") || (code != ExitOK && f.exists(f.deployedJar("ux"))) {
			t.Errorf("%s: checked %v, calls:\n%s", tc.name, checked, f.calls())
		}
		if (tc.id == "" || tc.mode == "C") && strings.Contains(f.out, "cache-buster") {
			t.Errorf("%s: stdout:\n%s", tc.name, f.out)
		}
	}
}
