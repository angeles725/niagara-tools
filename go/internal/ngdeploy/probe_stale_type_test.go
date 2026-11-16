package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

// A <jar>.ng-probe that is not a regular file was not left by a run (a probe moves a jar, a regular file):
// it is not moved to the jar's name. The run stops with exit 40 and names what it is, nothing changed
// (review T15 R3 stale-probe-any-type).
func TestAProbeNameThatIsNotARegularFileIsRefusedAndLeftAlone(t *testing.T) {
	for kind, make := range map[string]func(f *fixture, probe string) error{
		"a directory":     func(_ *fixture, probe string) error { return os.Mkdir(probe, 0o755) },
		"a symbolic link": func(f *fixture, probe string) error { return os.Symlink(f.dir+"/elsewhere.jar", probe) },
	} {
		f := newFixture(t)
		upgrade(t, f)
		jar := f.deployedJar("rt")
		if err := os.Remove(jar); err != nil {
			t.Fatal(err)
		}
		if err := make(f, jar+probeSuffix); err != nil {
			t.Fatal(err)
		}
		before := tree(t, f.dir)
		code := f.run("--no-backup")
		want := "[ng-deploy] ERROR copy refused: " + jar + probeSuffix + " is " + kind + ", not a jar a run left while it tested " +
			jar + ": move it out of the station and deploy again; nothing was built or copied\n"
		if code != ExitCopy || f.err != want || f.out != "" || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%s: exit %d, stdout %q, stderr %q, calls %q", kind, code, f.out, f.err, f.calls())
		}
		if after := tree(t, f.dir); after != before {
			t.Errorf("%s: the tree changed:\n%s", kind, after)
		}
	}
}
