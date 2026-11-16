package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

func TestVersionPrecheck(t *testing.T) {
	for _, tc := range []struct {
		name, old, built string
		code             int
		want             string // a stdout part, or with "err:" the ERROR text
	}{
		{"a higher version deploys", "1.3", "1.4", ExitOK, "[ng-deploy] version-bump ok (demo-rt.jar: 1.3 -> 1.4)\n"},
		{"numeric per component", "1.9", "1.10", ExitOK, "[ng-deploy] version-bump ok (demo-rt.jar: 1.9 -> 1.10)\n"},
		{"the same version", "2.1.0", "2.1.0", ExitVersionSame, "err:version-bump precheck: demo-rt.jar bytes changed but vendorVersion is still 2.1.0 — bump defaultModuleVersion and rebuild"},
		{"a trailing zero is not a bump", "1.3", "1.3.0", ExitVersionSame, "err:version-bump precheck: demo-rt.jar bytes changed but vendorVersion is still 1.3.0 — bump defaultModuleVersion and rebuild"},
		{"a lower version", "1.10", "1.9", ExitVersionDown, "err:version-bump precheck: demo-rt.jar vendorVersion went down (1.10 -> 1.9) — a deploy must raise it; bump defaultModuleVersion above 1.10 and rebuild"},
		{"suffixes that differ", "1.3", "1.3-rc", ExitVersionUnknown, "err:version-bump precheck cannot verify demo-rt.jar: vendorVersions are not comparable (built='1.3-rc' deployed='1.3': their non-numeric suffixes differ)"},
		{"both unreadable", "", "", ExitVersionUnknown, "err:version-bump precheck cannot verify demo-rt.jar: vendorVersion unreadable (built='' deployed='')"},
		{"deployed unreadable", "", "1.2", ExitVersionUnknown, "err:version-bump precheck cannot verify demo-rt.jar: vendorVersion unreadable (built='1.2' deployed='')"},
		{"built unreadable", "1.2", "", ExitVersionUnknown, "err:version-bump precheck cannot verify demo-rt.jar: vendorVersion unreadable (built='' deployed='1.2')"},
	} {
		f := newFixture(t)
		f.zipJar(f.deployedJar("rt"), builtAt("1788609301540", nil)) // real jars: with the same version their content is read
		f.zipJar(f.builtJar("rt"), builtAt("1788609309999", map[string]string{"com/acme/BDemo.class": "changed"}))
		t.Setenv("FAKE_OLD_VERSION", tc.old)
		t.Setenv("FAKE_NEW_VERSION", tc.built)
		code := f.run("--no-backup", "--mode", "C")
		want := tc.want
		if rest, ok := strings.CutPrefix(want, "err:"); ok {
			want = "err:" + warnNoBackup + "[ng-deploy] ERROR " + rest + "\n"
		}
		f.want(code, tc.code, want)
		if deployedNow := f.read(f.deployedJar("rt")); (deployedNow == f.read(f.builtJar("rt"))) != (tc.code == ExitOK) {
			t.Errorf("%s: the station jar is the built one: %v, after exit %d", tc.name, deployedNow == f.read(f.builtJar("rt")), code)
		}
		if tc.code != ExitOK && strings.Contains(f.calls(), "vm.sh") {
			t.Errorf("%s: the gate ran after a refused precheck", tc.name)
		}
	}
}

func TestVersionPrecheckSkipsAJarThatIsNewOrUnchanged(t *testing.T) {
	f := newFixture(t) // nothing deployed
	f.want(f.run("--no-backup"), ExitOK)
	if strings.Contains(f.out, "version-bump") {
		t.Errorf("stdout:\n%s", f.out)
	}
	f.reset() // now both jars are deployed with the same bytes: no version is read
	f.want(f.run("--no-backup"), ExitOK)
	if strings.Contains(f.out, "version-bump") || strings.Count(f.calls(), "unzip") != 2 {
		t.Errorf("stdout:\n%scalls:\n%s", f.out, f.calls())
	}
}

func TestVersionPrecheckChecksEveryJarOfTheMode(t *testing.T) {
	f := newFixture(t)
	f.write(f.deployedJar("rt"), "rtjar") // unchanged
	f.zipJar(f.deployedJar("ux"), builtAt("1788609301540", nil))
	f.zipJar(f.builtJar("ux"), builtAt("1788609309999", map[string]string{"com/acme/BDemo.class": "changed"}))
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.0")
	f.want(f.run("--no-backup"), ExitVersionSame, "err:demo-ux.jar bytes changed but vendorVersion is still 1.0")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK) // the ux jar is not in scope
}

// Declared difference: bash read "cannot check" as "no deployed jar" and skipped the precheck.
func TestVersionPrecheckRefusesAJarItCannotCheck(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root checks everything")
	}
	f := newFixture(t)
	f.deployed()
	f.chmod(f.modules, 0o000)
	code := f.run("--no-backup", "--mode", "C")
	want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck cannot verify demo-rt.jar: cannot check " + f.modules + "/demo-rt.jar: Permission denied\n"
	if code != ExitVersionUnknown || f.err != want {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

// A build that passed and left no jar is refused before any jar is compared, deployed or not.
func TestVersionPrecheckRefusesABuildThatLeftNoJar(t *testing.T) {
	for name, arrange := range map[string]func(f *fixture){
		"nothing deployed": func(*fixture) {},
		"an older deploy":  func(f *fixture) { f.deployed() },
		"a directory":      func(f *fixture) { f.mkdir(f.builtJar("ux")) },
	} {
		f := newFixture(t)
		f.remove(f.builtJar("ux"))
		arrange(f)
		before := f.read(f.deployedJar("rt"))
		code := f.run("--no-backup")
		want := warnNoBackup + "[ng-deploy] ERROR version-bump precheck cannot verify demo-ux.jar: the build left no jar at " + f.builtJar("ux") + "\n"
		if code != ExitVersionUnknown || f.err != want || !strings.HasSuffix(f.out, "[ng-deploy] build ok\n") {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", name, code, f.out, f.err)
		}
		if strings.Contains(f.calls(), "unzip") || strings.Contains(f.calls(), "sr.sh") || f.read(f.deployedJar("rt")) != before || f.exists(f.work+"/.last-deploy-sha") {
			t.Errorf("%s: the deploy went past the precheck:\n%s", name, f.calls())
		}
	}
}
