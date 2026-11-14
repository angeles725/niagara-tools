package ngdeploy

import (
	"os"
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

// Kept from the bash tool (set -e over `unzip | grep -c`): a jar with no type entry ends the deploy with
// exit 1 and no message, even when 0 was expected; a failed unzip that still listed entries ends it with
// unzip's own status.
func TestAModuleXMLThatCannotBeCountedEndsTheDeploy(t *testing.T) {
	f := newFixture(t)
	f.set("EXPECTED_UX_TYPES", "0")
	t.Setenv("FAKE_UNZIP_TYPES", "0")
	if code := f.run("--no-backup", "--mode", "B"); code != ExitFault || f.err != warnNoBackup || f.exists(f.deployedJar("ux")) {
		t.Errorf("no type entry: exit %d, stderr %q", code, f.err)
	}
	f = newFixture(t)
	t.Setenv("FAKE_UNZIP_EXIT", "9")
	t.Setenv("FAKE_UNZIP_ERR", "cannot find zipfile")
	if code := f.run("--no-backup", "--mode", "C"); code != 9 || f.err != warnNoBackup+"unzip: cannot find zipfile\n" || f.exists(f.deployedJar("rt")) {
		t.Errorf("failed unzip: exit %d, stderr %q", code, f.err)
	}
	f = newFixture(t)
	if err := os.Remove(f.bin + "/unzip"); err != nil {
		t.Fatal(err)
	}
	if code := f.run("--no-backup", "--mode", "C"); code != ExitFault || f.err != warnNoBackup+"ng-deploy.sh: unzip: command not found\n" || f.exists(f.deployedJar("rt")) {
		t.Errorf("no unzip: exit %d, stderr %q", code, f.err)
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
