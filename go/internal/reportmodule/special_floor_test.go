package reportmodule

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// schemaRiskRow runs the schema-risk step on dir with no baseline and returns the row it printed and whether
// the run counts as floor-incomplete.
func schemaRiskRow(t *testing.T, dir string, firstDeploy bool) (row string, floorSkip bool) {
	t.Helper()
	var out bytes.Buffer
	r := &report{out: &out, o: options{firstDeploy: firstDeploy}, n: map[string]int{}}
	r.schemaRisk(artifact{"Mod-rt", dir})
	return strings.TrimSuffix(out.String(), "\n"), r.floorSkip
}

// TestSchemaRiskNoBaselineNamesTheCommand pins B6: a missing .deploy-baseline names the exact command that
// creates it from the deployed commit, the row stays a floor SKIP (exit 4 INCOMPLETE, build.sh 53), and the
// command, run as printed with the ref filled in, extracts the artifact as it was at that ref.
func TestSchemaRiskNoBaselineNamesTheCommand(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git")
	}
	repo := t.TempDir()
	dir := filepath.Join(repo, "Mod", "Mod rt") // a space: the command must quote the path
	must(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	must(t, os.WriteFile(filepath.Join(dir, "module-include.xml"), []byte("<deployed/>"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "src", "A.java"), []byte("class A {}"), 0o644))
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@t", "-c",
			"commit.gpgsign=false"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("add", ".")
	git("commit", "-q", "-m", "deployed")
	git("tag", "deployed-2.8.5")
	must(t, os.WriteFile(filepath.Join(dir, "module-include.xml"), []byte("<work-tree/>"), 0o644))

	row, floorSkip := schemaRiskRow(t, dir, false)
	if !floorSkip {
		t.Errorf("a missing baseline must stay a floor SKIP (INCOMPLETE)")
	}
	quoted := "'" + dir + "'"
	want := "Mod-rt  SKIP  schema-risk  no .deploy-baseline (floor check not run); create it from the deployed commit" +
		" (put its tag or commit in place of <deployed-ref>): " +
		`T=$(mktemp -d) && git -C ` + quoted + ` archive -o "$T/b.tar" <deployed-ref> . && mkdir ` + quoted +
		`/.deploy-baseline && tar -x -f "$T/b.tar" -C ` + quoted + `/.deploy-baseline; rm -rf "$T"` +
		" — or --first-deploy for a never-deployed module (build-verify.md § Schema-risk deploy workflow)"
	if row != want {
		t.Fatalf("row:\n got %q\nwant %q", row, want)
	}
	command := row[strings.Index(row, "T=$(mktemp -d)"):strings.Index(row, " — or --first-deploy")]
	run := func(ref string) {
		t.Helper()
		cmd := exec.Command("bash", "-c", strings.ReplaceAll(command, "<deployed-ref>", ref))
		cmd.Dir = t.TempDir() // a pasted redirection must not touch the repo
		_ = cmd.Run()
	}
	baseline := filepath.Join(dir, ".deploy-baseline")
	// A wrong ref, or <deployed-ref> pasted as it is printed, creates no (empty) baseline.
	for _, ref := range []string{"no-such-ref", "<deployed-ref>"} {
		run(ref)
		if _, err := os.Stat(baseline); err == nil {
			t.Fatalf("ref %q: a failed archive left %s", ref, baseline)
		}
	}
	run("deployed-2.8.5")
	got, err := os.ReadFile(filepath.Join(baseline, "module-include.xml"))
	must(t, err)
	if string(got) != "<deployed/>" {
		t.Errorf("baseline module-include.xml = %q, want the deployed commit's", got)
	}
	if _, err := os.Stat(filepath.Join(baseline, "src", "A.java")); err != nil {
		t.Errorf("baseline lacks src/A.java: %v", err)
	}
}

// TestSchemaRiskFirstDeployUnchanged pins the --first-deploy row as it was: N/A, not a floor SKIP.
func TestSchemaRiskFirstDeployUnchanged(t *testing.T) {
	t.Parallel()
	row, floorSkip := schemaRiskRow(t, t.TempDir(), true)
	if want := "Mod-rt  SKIP  schema-risk  N/A (--first-deploy: never deployed, no baseline to diff)"; row != want || floorSkip {
		t.Errorf("row %q floorSkip %v; want %q false", row, floorSkip, want)
	}
}

func TestShellWord(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{"/a/Mod-rt": "/a/Mod-rt", "m/N1": "m/N1", "/a b": "'/a b'",
		"/it's": `'/it'\''s'`, "": "''", "/x$y": "'/x$y'"} {
		if got := ShellWord(in); got != want {
			t.Errorf("ShellWord(%q) = %q, want %q", in, got, want)
		}
	}
}
