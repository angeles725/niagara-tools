package buildmodule

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// git runs git in dir for a fixture repository, failing the test when it fails.
func (f *fixture) git(dir string, args ...string) string {
	f.t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t",
		"-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// deployedRepo makes the fixture's module root a git repository whose tag v1 holds every artifact with a
// marker file reading "v1"; the work tree then reads "v2", so a report that sees "v1" saw the tag's tree.
func (f *fixture) deployedRepo() {
	f.t.Helper()
	f.git(f.root, "init", "-q")
	for _, p := range []string{"rt", "ux", "wb"} {
		f.write(f.artifact(p)+"/marker", "v1")
	}
	f.git(f.root, "add", "-A")
	f.git(f.root, "commit", "-q", "-m", "deployed")
	f.git(f.root, "tag", "v1")
	for _, p := range []string{"rt", "ux", "wb"} {
		f.write(f.artifact(p)+"/marker", "v2")
	}
}

func (f *fixture) artifact(profile string) string { return f.root + "/Foo/Foo-" + profile }

// seesBaselines makes the report stub log, per artifact, the marker its .deploy-baseline holds and where the
// baseline lives, then exit with code.
func (f *fixture) seesBaselines(code string) {
	f.does(reportScript, `for a in "$1"/*/; do a=${a%/}; n=${a##*/}
  if [ -d "$a/.deploy-baseline" ]; then echo "$n $(cat "$a/.deploy-baseline/marker") $(readlink "$a/.deploy-baseline")" >> "`+
		f.dir+`/seen.log"; else echo "$n none" >> "`+f.dir+`/seen.log"; fi; done
exit `+code)
}

// seen is what the report stub logged per artifact: name -> "marker target" or "none".
func (f *fixture) seen() map[string]string {
	got := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(f.read(f.dir+"/seen.log")), "\n") {
		if name, rest, ok := strings.Cut(line, " "); ok {
			got[name] = rest
		}
	}
	return got
}

// noBaselineLeft fails unless no artifact holds a .deploy-baseline (not even a dangling link) and every
// extraction the report saw is gone.
func (f *fixture) noBaselineLeft(name string) {
	f.t.Helper()
	for _, p := range []string{"rt", "ux", "wb"} {
		if _, err := os.Lstat(f.artifact(p) + "/.deploy-baseline"); err == nil {
			f.t.Errorf("%s: Foo-%s/.deploy-baseline is left after the run", name, p)
		}
	}
	for art, rest := range f.seen() {
		if _, target, ok := strings.Cut(rest, " "); ok && target != "" {
			if _, err := os.Stat(target); err == nil {
				f.t.Errorf("%s: the extraction of %s is left after the run: %s", name, art, target)
			}
		}
	}
}

// TestSchemaBaselineRefFeedsTheReport: --schema-baseline-ref <ref> extracts every artifact as it is at <ref>
// to where report-module reads its schema-risk baseline, so the floor check runs and the run ends 0.
func TestSchemaBaselineRefFeedsTheReport(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.deployedRepo()
	f.seesBaselines("0")
	f.wants("ref", f.build("--schema-baseline-ref", "v1"), ExitOK,
		"==> schema-risk baseline: Foo-rt from v1", "==> schema-risk baseline: Foo-ux from v1",
		"==> schema-risk baseline: Foo-wb from v1")
	seen := f.seen()
	for _, p := range []string{"rt", "ux", "wb"} {
		if got := seen["Foo-"+p]; !strings.HasPrefix(got, "v1 /") {
			t.Errorf("Foo-%s: the report saw %q, want the v1 tree through a link to a temporary directory", p, got)
		}
	}
	if strings.Contains(f.err, "INCOMPLETE") {
		t.Errorf("the report is still INCOMPLETE: %s", f.err)
	}
	f.noBaselineLeft("ref")
}

// TestSchemaBaselineRefIsRemovedWhateverTheExit: the extraction and its link go when the run ends, also when
// the report is INCOMPLETE (exit 53).
func TestSchemaBaselineRefIsRemovedWhateverTheExit(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.deployedRepo()
	f.seesBaselines("4")
	f.wants("incomplete", f.build("--schema-baseline-ref", "v1"), ExitReportPartial, msgReportIncomplete)
	if len(f.seen()) != 3 {
		t.Fatalf("the report did not run on 3 artifacts: %v", f.seen())
	}
	f.noBaselineLeft("incomplete")
}

// TestSchemaBaselineRefRefusals: a ref that is not a commit, a module directory absent at the ref, an artifact
// that already has a .deploy-baseline and --first-deploy are usage errors (exit 2) before gradle, and the
// module tree is left as it was.
func TestSchemaBaselineRefRefusals(t *testing.T) {
	for _, c := range []struct {
		name, ref, want string
		flags           []string
		prepare         func(f *fixture)
	}{
		{"not a commit", "nope", "build.sh: --schema-baseline-ref nope: not a commit of the repository holding ", nil, nil},
		{"module absent at the ref", "v0", "build.sh: --schema-baseline-ref v0: ", nil, func(f *fixture) {}},
		{"a baseline is already there", "v1", "Foo-ux/.deploy-baseline already exists", nil, func(f *fixture) {
			f.write(f.artifact("ux")+"/.deploy-baseline/marker", "mine")
		}},
		{"--first-deploy", "v1", "build.sh: --schema-baseline-ref and --first-deploy exclude each other", []string{"--first-deploy"}, nil},
	} {
		f := newFixture(t)
		f.signs()
		if c.ref == "v0" { // a commit before the module existed
			f.git(f.dir, "init", "-q", f.root)
			f.write(f.root+"/README", "x")
			f.git(f.root, "add", "README")
			f.git(f.root, "commit", "-q", "-m", "before")
			f.git(f.root, "tag", "v0")
		}
		f.deployedRepo()
		if c.prepare != nil {
			c.prepare(f)
		}
		f.seesBaselines("0")
		got := f.build(append(c.flags, "--schema-baseline-ref", c.ref)...)
		f.wants(c.name, got, ExitUsage, c.want)
		if c.ref == "v0" && !strings.Contains(f.err, "/Foo does not exist at that commit") {
			t.Errorf("%s: the refusal does not name the module directory: %s", c.name, f.err)
		}
		if f.call("gradlew") != "" || f.call(reportScript) != "" {
			t.Errorf("%s: the chain went on after the refusal:\n%s", c.name, f.calls())
		}
		if c.prepare != nil && c.ref == "v1" && f.read(f.artifact("ux")+"/.deploy-baseline/marker") != "mine" {
			t.Errorf("%s: the existing baseline was changed", c.name)
		}
		for _, p := range []string{"rt", "wb"} {
			if _, err := os.Lstat(f.artifact(p) + "/.deploy-baseline"); err == nil {
				t.Errorf("%s: Foo-%s/.deploy-baseline was made", c.name, p)
			}
		}
	}
}

// TestSchemaBaselineRefSkipsAnArtifactNewSinceTheRef: an artifact added after the ref gets no baseline (one
// INFO line; its schema-risk row stays the SKIP it was), the others get theirs.
func TestSchemaBaselineRefSkipsAnArtifactNewSinceTheRef(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.deployedRepo()
	f.write(f.root+"/Foo/Foo-se/marker", "new")
	f.write(f.root+"/Foo/Foo-se/Foo-se.gradle.kts", "") // a real part: its build file makes it an artifact (K4)
	f.seesBaselines("4")
	f.wants("new artifact", f.build("--schema-baseline-ref", "v1"), ExitReportPartial,
		"build.sh: INFO — Foo-se does not exist at v1: no schema-risk baseline for it (a first deploy of that artifact)",
		msgReportIncomplete+msgSchemaRiskCrossRef)
	seen := f.seen()
	if seen["Foo-se"] != "none" || !strings.HasPrefix(seen["Foo-rt"], "v1 ") {
		t.Errorf("the report saw %v", seen)
	}
	f.noBaselineLeft("new artifact")
}

// TestNoSchemaBaselineRefLeavesTheTreeAlone: without the option nothing is extracted and git is never asked:
// the report sees no baseline, as before.
func TestNoSchemaBaselineRefLeavesTheTreeAlone(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.deployedRepo()
	f.seesBaselines("4")
	f.wants("no option", f.build(), ExitReportPartial, msgReportIncomplete+msgSchemaRiskCrossRef)
	for _, p := range []string{"rt", "ux", "wb"} {
		if got := f.seen()["Foo-"+p]; got != "none" {
			t.Errorf("Foo-%s: the report saw a baseline %q without the option", p, got)
		}
	}
	if strings.Contains(f.out, "schema-risk baseline") {
		t.Errorf("a baseline line was printed without the option: %s", f.out)
	}
}
