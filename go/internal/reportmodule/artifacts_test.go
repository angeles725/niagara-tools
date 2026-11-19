package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stubbedRun runs report-module (not --lint-only) on root with every toolbelt member a no-op stub that exits 0,
// and returns its exit code and stdout.
func stubbedRun(t *testing.T, root string) (int, string) {
	t.Helper()
	repo, err := filepath.Abs("../../..")
	must(t, err)
	dir := t.TempDir()
	tb := filepath.Join(dir, "toolbelt")
	must(t, os.MkdirAll(tb, 0o755))
	must(t, os.WriteFile(filepath.Join(tb, ".stub-rest"), nil, 0o644))
	toolbeltCopy(t, repo, dir)
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		return os.Getenv(k)
	}}
	return mainEnv([]string{root}, s, os.Environ()), o.String()
}

// builtPart is the files of a built module part with its deployed baseline: its build file, a jar under
// build/libs and a .deploy-baseline directory.
func builtPart(files map[string]string, name string) {
	files[name+"/"+name+".gradle.kts"] = ""
	files[name+"/build/libs/"+name+".jar"] = "jar"
	files[name+"/.deploy-baseline/module-include.xml"] = "<deployed/>"
}

// TestDocsDirectoryIsNoArtifact pins K4 (b): a docs/ directory beside built parts is no artifact. It gets one
// INFO line, no verify-module or schema-risk row (so no floor SKIP), and the run is not INCOMPLETE because of it;
// the real parts keep their schema-risk rows.
func TestDocsDirectoryIsNoArtifact(t *testing.T) {
	files := map[string]string{"docs/README.md": "plain notes"}
	builtPart(files, "Foo-rt")
	builtPart(files, "Foo-ux")
	code, out := stubbedRun(t, moduleTree(t, files))
	if code != ExitClean {
		t.Errorf("exit %d, want %d\n%s", code, ExitClean, out)
	}
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, "docs  ") && !strings.HasPrefix(ln, "docs  INFO  artifact  ") {
			t.Errorf("docs/ got a row: %q", ln)
		}
	}
	for _, want := range []string{
		"docs  INFO  artifact  not a module artifact (no module-include.xml, not a module part with its build file): skipped\n",
		"Foo-rt  PASS  schema-risk  verdict=SAFE\n", "Foo-ux  PASS  schema-risk  verdict=SAFE\n",
		"report-module: 2 artifacts · ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "INCOMPLETE") {
		t.Errorf("the run is INCOMPLETE:\n%s", out)
	}
}

// TestNoRecognizableArtifactWalksEveryDirectory: a module root where no directory reads as an artifact keeps the
// walk it had (every directory, no INFO line), so nothing is silently dropped from the report.
func TestNoRecognizableArtifactWalksEveryDirectory(t *testing.T) {
	_, out := stubbedRun(t, moduleTree(t, map[string]string{"A/x": "", "docs/README.md": ""}))
	for _, want := range []string{"A  SKIP  schema-risk  ", "docs  SKIP  schema-risk  ", "report-module: 2 artifacts · "} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "INFO  artifact") {
		t.Errorf("an INFO artifact line without a recognizable artifact:\n%s", out)
	}
}

// TestDocsWithABaselineIsNoArtifact (verifier probe, S1 real shape): a docs/ folder that even holds a
// .deploy-baseline is no artifact, so schema-risk never runs on it; the one real part makes the run clean.
func TestDocsWithABaselineIsNoArtifact(t *testing.T) {
	files := map[string]string{"docs/.deploy-baseline/README.md": "old notes", "docs/README.md": "notes"}
	builtPart(files, "Foo-rt")
	code, out := stubbedRun(t, moduleTree(t, files))
	if code != ExitClean {
		t.Errorf("exit %d, want %d\n%s", code, ExitClean, out)
	}
	if strings.Contains(out, "docs  PASS  schema-risk") || strings.Contains(out, "docs  SKIP  schema-risk") ||
		strings.Contains(out, "docs  ERROR  schema-risk") {
		t.Errorf("docs/ got a schema-risk row:\n%s", out)
	}
	if !strings.Contains(out, "report-module: 1 artifact · ") {
		t.Errorf("want 1 artifact:\n%s", out)
	}
}

// TestOtherFolderBesideThePartsIsNoArtifact (verifier probe): any non-part folder (openspec/) is skipped like
// docs/: one INFO line, not counted.
func TestOtherFolderBesideThePartsIsNoArtifact(t *testing.T) {
	files := map[string]string{"openspec/spec.md": "spec"}
	builtPart(files, "Foo-rt")
	code, out := stubbedRun(t, moduleTree(t, files))
	if code != ExitClean {
		t.Errorf("exit %d, want %d\n%s", code, ExitClean, out)
	}
	for _, want := range []string{"openspec  INFO  artifact  " + NotArtifactDetail + ": skipped\n", "report-module: 1 artifact · "} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
}

// TestGradleOnlyPartBesideDocsIsAnArtifact (verifier probe): a -wb part with only its <name>.gradle.kts (no
// module-include.xml) is still an artifact beside docs/: 2 artifacts, its rows present.
func TestGradleOnlyPartBesideDocsIsAnArtifact(t *testing.T) {
	files := map[string]string{"docs/README.md": "notes", "Foo-wb/Foo-wb.gradle.kts": ""}
	builtPart(files, "Foo-rt")
	_, out := stubbedRun(t, moduleTree(t, files))
	for _, want := range []string{"report-module: 2 artifacts · ", "Foo-wb  SKIP  schema-risk  ", "docs  INFO  artifact  "} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q\n%s", want, out)
		}
	}
}
