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
	if err := os.Symlink(f.dir+"/gone/baseline", link); err != nil {
		t.Fatal(err)
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
