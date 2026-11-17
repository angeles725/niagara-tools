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
