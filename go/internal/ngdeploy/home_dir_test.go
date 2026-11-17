package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// A NIAGARA_HOME that names no directory, or names a file, is refused with exit 10 before the backup and
// before gradle runs, with a line that names the value; nothing in the station changes (bench Δ25).
func TestANiagaraHomeThatIsNotADirectoryIsRefusedBeforeGradle(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture) string
		want    string
	}{
		{"no directory there", func(f *fixture) string { return f.dir + "/no such home" }, msgHomeMissing},
		{"a file", func(f *fixture) string {
			f.write(f.dir+"/home-file", "not a directory")
			return f.dir + "/home-file"
		}, msgHomeNotADir},
	} {
		f := newFixture(t)
		f.deployed()
		home := tc.arrange(f)
		f.set("NIAGARA_HOME", `"`+home+`"`)
		code := f.run()
		line := "[ng-deploy] ERROR " + strings.Replace(tc.want, "%s", home, 1) + "\n"
		if code != ExitEnv || f.err != line {
			t.Errorf("%s: exit %d, stderr %q, want %q", tc.name, code, f.err, line)
		}
		if f.exists(f.dir + "/calls.log") {
			t.Errorf("%s: a command ran:\n%s", tc.name, f.calls())
		}
		if f.read(f.deployedJar("rt")) != "old-rt" {
			t.Errorf("%s: the station changed:\n%s", tc.name, tree(t, f.modules))
		}
		if entries, _ := os.ReadDir(f.modules); len(entries) != 2 {
			t.Errorf("%s: the station holds:\n%s", tc.name, tree(t, f.modules))
		}
	}
}

// A NIAGARA_HOME reached through a symbolic link to a directory is a directory: the deploy goes through.
func TestANiagaraHomeThroughALinkToADirectoryDeploys(t *testing.T) {
	f := newFixture(t)
	if err := os.Symlink(f.home, f.dir+"/home-link"); err != nil {
		t.Fatal(err)
	}
	f.set("NIAGARA_HOME", f.dir+"/home-link")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
}

// A NIAGARA_HOME under a directory the deploy may not search is a permission error, said as one, not a
// home that is not there; exit 10 before gradle all the same.
func TestANiagaraHomeThatCannotBeLookedAtSaysWhy(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root searches every directory")
	}
	f := newFixture(t)
	home := f.dir + "/locked/home"
	f.mkdir(home)
	if err := os.Chmod(f.dir+"/locked", 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(f.dir+"/locked", 0o755) })
	f.set("NIAGARA_HOME", `"`+home+`"`)
	code := f.run("--no-backup")
	line := "[ng-deploy] ERROR NIAGARA_HOME " + home + " cannot be looked at (Permission denied): gradle builds against <NIAGARA_HOME> and installs the jars in its modules/; " +
		"let the deploy search every directory of its path, or point it elsewhere; nothing was built or changed\n"
	if code != ExitEnv || f.err != line || f.exists(f.dir+"/calls.log") {
		t.Errorf("exit %d, stderr %q, want %q", code, f.err, line)
	}
}

// A relative NIAGARA_HOME reaches gradle as an absolute path, as build.sh hands it (gradle runs in the gradle
// root, where the relative path names another directory); the lines of the deploy name it as given.
func TestARelativeNiagaraHomeReachesGradleAbsolute(t *testing.T) {
	f := newFixture(t)
	f.set("NIAGARA_HOME", ".") // the fixture runs in f.work, a directory that is not a mirror
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] copy ok\n")
	if !strings.Contains(f.calls(), "gradlew <-Pniagara_home="+f.work+"> ") {
		t.Errorf("gradle got:\n%s", f.calls())
	}
	f = newFixture(t)
	f.ownHome = true // no --install-into-home: the refusal names the home as given
	f.set("NIAGARA_HOME", ".")
	if code := f.run("--no-backup"); code != ExitEnv || !strings.HasPrefix(f.err, "[ng-deploy] ERROR NIAGARA_HOME . is a real niagara_home") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}
