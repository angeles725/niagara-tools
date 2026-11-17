package buildmodule

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestSchemaBaselineRefReplacesADanglingLeftover: the dangling .deploy-baseline link a killed run leaves is
// removed with one INFO line and the run takes the ref, instead of refusing a retry.
func TestSchemaBaselineRefReplacesADanglingLeftover(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.deployedRepo()
	link := f.artifact("ux") + "/.deploy-baseline"
	// The leftover is a link a real run made: the first run's report copies it, and it is put back after that
	// run removed it and its extraction, as a SIGKILL would have left it.
	f.does(reportScript, `cp -P "$1/Foo-ux/.deploy-baseline" "`+f.dir+`/kept-link"`)
	f.wants("first run", f.build("--schema-baseline-ref", "v1"), ExitOK)
	if err := os.Rename(f.dir+"/kept-link", link); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(link); err == nil {
		t.Fatalf("the kept link is not dangling: %s", link)
	}
	f.seesBaselines("0")
	f.wants("leftover", f.build("--schema-baseline-ref", "v1"), ExitOK,
		"build.sh: INFO — removed "+link+", a dangling link a killed --schema-baseline-ref run left",
		"==> schema-risk baseline: Foo-ux from v1")
	if got := f.seen()["Foo-ux"]; !strings.HasPrefix(got, "v1 /") {
		t.Errorf("Foo-ux: the report saw %q, want the v1 tree", got)
	}
	f.noBaselineLeft("leftover")
}

// TestSchemaBaselineRefKeepsOperatorLinks: a link that is not a killed run's leftover (a live link, a dangling
// link not shaped like this tool's extraction) is refused like a directory and left in place; a refusal also
// leaves a leftover on another artifact in place (a refusal changes nothing).
func TestSchemaBaselineRefKeepsOperatorLinks(t *testing.T) {
	for _, c := range []struct{ name, target string }{
		{"live link", "/live"}, {"dangling operator link", "/share/baseline"},
	} {
		f := newFixture(t)
		f.signs()
		f.deployedRepo()
		f.mkdir(f.dir + "/live")
		ux, rt := f.artifact("ux")+"/.deploy-baseline", f.artifact("rt")+"/.deploy-baseline"
		f.link(f.dir+c.target, ux)
		f.link(f.dir+"/tmp.gone/baseline", rt)
		f.wants(c.name, f.build("--schema-baseline-ref", "v1"), ExitUsage, "Foo-ux/.deploy-baseline already exists")
		for link, target := range map[string]string{ux: f.dir + c.target, rt: f.dir + "/tmp.gone/baseline"} {
			if got, err := os.Readlink(link); err != nil || got != target {
				t.Errorf("%s: %s now reads %q (%v), want it kept -> %s", c.name, link, got, err, target)
			}
		}
		if strings.Contains(f.err, "INFO — removed") || f.call("gradlew") != "" {
			t.Errorf("%s: the refusal changed something:\n%s\n%s", c.name, f.err, f.calls())
		}
	}
}

// link makes a symlink at path pointing to target.
func (f *fixture) link(target, path string) {
	f.t.Helper()
	if err := os.Symlink(target, path); err != nil {
		f.t.Fatal(err)
	}
}

// TestSchemaBaselineRefToolFailures: git that cannot be started is exit 10 naming it; an extraction that fails
// (here tar) is exit 1 naming the artifact. Both end before gradle and leave no .deploy-baseline.
func TestSchemaBaselineRefToolFailures(t *testing.T) {
	path := os.Getenv("PATH")
	for _, c := range []struct {
		name  string
		tools map[string]string // PATH entries: name -> real command to link, "" for a stub that fails
		code  int
		want  string
	}{
		{"git missing", map[string]string{}, ExitEnv, "build.sh: git: command not found"},
		{"tar fails", map[string]string{"git": "git", "mktemp": "mktemp", "mkdir": "mkdir", "tar": ""}, ExitFault,
			"build.sh: --schema-baseline-ref v1: cannot extract "},
	} {
		t.Setenv("PATH", path) // the previous case narrowed it
		f := newFixture(t)
		f.signs()
		f.deployedRepo()
		bin := f.dir + "/bin"
		f.mkdir(bin)
		for name, real := range c.tools {
			if real == "" {
				f.write(bin+"/"+name, "#!/bin/sh\nexit 2\n")
				if err := os.Chmod(bin+"/"+name, 0o755); err != nil {
					t.Fatal(err)
				}
				continue
			}
			target, err := exec.LookPath(real)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, bin+"/"+name); err != nil {
				t.Fatal(err)
			}
		}
		t.Setenv("PATH", bin)
		f.wants(c.name, f.build("--schema-baseline-ref", "v1"), c.code, c.want)
		if f.call("gradlew") != "" {
			t.Errorf("%s: gradle ran after the failure", c.name)
		}
		f.noBaselineLeft(c.name)
	}
}
