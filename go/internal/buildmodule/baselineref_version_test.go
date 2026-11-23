package buildmodule

import (
	"strings"
	"testing"
)

// versionedRepo is deployedRepo with the group build file naming defaultModuleVersion(<atRef>) at tag v1 and the
// work tree naming 9.9.9.
func (f *fixture) versionedRepo(atRef string) {
	f.t.Helper()
	f.write(f.root+"/build.gradle.kts", "defaultModuleVersion(\""+atRef+"\")\n")
	f.deployedRepo()
	f.write(f.root+"/build.gradle.kts", "defaultModuleVersion(\"9.9.9\")\n")
}

// TestSchemaBaselineRefOlderThanTheDeployedJarWarns: a baseline of 2.3.0 with 2.3.1 deployed is checked against the
// wrong floor; the build still passes (exit 0) and the stdout lines are those of a run without the check.
func TestSchemaBaselineRefOlderThanTheDeployedJarWarns(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.versionedRepo("2.3.0")
	f.jar(f.deployed("rt"), "2.3.1", "x")
	f.seesBaselines("0")
	f.wants("older", f.build("--schema-baseline-ref", "v1"), ExitOK,
		"build.sh: WARN — --schema-baseline-ref v1 has defaultModuleVersion 2.3.0 but the deployed Foo-rt is 2.3.1: "+
			"the schema-risk baseline is not what runs on the station; pass the ref of the deployed version\n",
		"==> schema-risk baseline: Foo-rt from v1")
}

// TestSchemaBaselineRefMatchingTheDeployedJarIsSilent: equal versions (1.3 == 1.3.0) and a ref with no readable
// version add no WARN.
func TestSchemaBaselineRefMatchingTheDeployedJarIsSilent(t *testing.T) {
	for _, c := range []struct{ name, atRef, deployed string }{
		{"equal", "2.3.1", "2.3.1"}, {"missing component is 0", "1.3", "1.3.0"}, {"no version at the ref", "", "2.3.1"},
	} {
		f := newFixture(t)
		f.signs()
		f.versionedRepo(c.atRef)
		if c.atRef == "" {
			f.write(f.root+"/build.gradle.kts", "")
			f.git(f.root, "add", "-A")
			f.git(f.root, "commit", "-q", "-m", "empty")
			f.git(f.root, "tag", "-f", "v1")
		}
		f.jar(f.deployed("rt"), c.deployed, "x")
		f.seesBaselines("0")
		if code := f.build("--schema-baseline-ref", "v1"); code != ExitOK || strings.Contains(f.err, "has defaultModuleVersion") {
			t.Errorf("%s: exit %d, stderr:\n%s", c.name, code, f.err)
		}
	}
}
