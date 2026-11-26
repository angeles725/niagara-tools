package buildmodule

import (
	"strings"
	"testing"
)

// targetRepo is versionedRepo with a .deploy-targets file in the module root holding line, where %SHA% is
// the commit tagged v1 (the deployed ref).
func (f *fixture) targetRepo(atRef, line string) {
	f.t.Helper()
	f.versionedRepo(atRef)
	f.write(f.root+"/"+deployTargetsFile, "# recorded deploys\n"+strings.ReplaceAll(line, "%SHA%", f.git(f.root, "rev-parse", "v1"))+"\n")
}

// TestTargetReadsTheRecordedDeploy: --target <name> takes the schema-risk baseline from the commit recorded for
// that target in .deploy-targets, never from the mirror: a mirror build of another version adds no WARN.
func TestTargetReadsTheRecordedDeploy(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.targetRepo("2.3.0", "plant1 Foo@2.3.0=%SHA%")
	f.jar(f.deployed("rt"), "2.4.0", "x") // my own mirror build of the next version
	f.seesBaselines("0")
	f.wants("target", f.build("--target", "plant1"), ExitOK,
		"==> deploy target plant1: Foo@2.3.0 at "+f.git(f.root, "rev-parse", "v1")+" (.deploy-targets)",
		"==> schema-risk baseline: Foo-rt from "+f.git(f.root, "rev-parse", "v1"))
	if got := f.seen()["Foo-rt"]; !strings.HasPrefix(got, "v1 /") {
		t.Errorf("the report saw %q, want the recorded commit's tree", got)
	}
	if strings.Contains(f.err, "WARN") || strings.Contains(f.err, "INCOMPLETE") {
		t.Errorf("stderr: %s", f.err)
	}
	f.noBaselineLeft("target")
}

// TestTargetWithoutARecordedDeployIsRefused: a target with no line for this module, or no file, or both options
// that name a baseline, is a usage error (exit 2) before gradle.
func TestTargetWithoutARecordedDeployIsRefused(t *testing.T) {
	for _, c := range []struct {
		name, line, want string
		flags            []string
	}{
		{"no line for the target", "plant2 Foo@2.3.0=%SHA%", "build.sh: --target plant1: no line for Foo in ", nil},
		{"no line for the module", "plant1 Bar@2.3.0=%SHA%", "no line for Foo in ", nil},
		{"a line that is not <MOD>@<version>=<sha>", "plant1 Foo=v1", "build.sh: --target plant1: cannot read ", nil},
		{"both options", "plant1 Foo@2.3.0=%SHA%", "build.sh: --target and --schema-baseline-ref/--first-deploy exclude each other",
			[]string{"--schema-baseline-ref", "v1"}},
	} {
		f := newFixture(t)
		f.signs()
		f.targetRepo("2.3.0", c.line)
		f.seesBaselines("0")
		f.wants(c.name, f.build(append([]string{"--target", "plant1"}, c.flags...)...), ExitUsage, c.want)
		if f.call("gradlew") != "" || f.call(reportScript) != "" {
			t.Errorf("%s: the chain went on after the refusal:\n%s", c.name, f.calls())
		}
	}
}

// TestTargetVersionDisagreeingWithItsCommitWarns: the recorded vendorVersion and the defaultModuleVersion at the
// recorded commit are two records of one deploy; when they differ the WARN says which is which, and the run goes on.
func TestTargetVersionDisagreeingWithItsCommitWarns(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.targetRepo("2.3.0", "plant1 Foo@2.3.5=%SHA%")
	f.seesBaselines("0")
	f.wants("disagree", f.build("--target", "plant1"), ExitOK,
		"build.sh: WARN — .deploy-targets records Foo@2.3.5 for plant1 but its commit declares defaultModuleVersion 2.3.0\n")
}
