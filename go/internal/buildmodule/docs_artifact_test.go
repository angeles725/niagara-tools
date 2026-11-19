package buildmodule

import (
	"strings"
	"testing"
)

// TestSchemaBaselineRefSkipsADocsDirectory pins K4 (a): a docs/ directory beside the module parts is no artifact
// (no module-include.xml, not a <MOD>-<part> with its build file), so --schema-baseline-ref extracts no
// baseline for it (one INFO line) while every real part still gets its own, and the run ends 0.
func TestSchemaBaselineRefSkipsADocsDirectory(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.write(f.root+"/Foo/docs/README.md", "plain notes")
	f.deployedRepo()
	f.seesBaselines("0")
	f.wants("docs", f.build("--schema-baseline-ref", "v1"), ExitOK,
		"build.sh: INFO — docs is not a module artifact (no module-include.xml, not a module part with its build file): no schema-risk baseline for it",
		"==> schema-risk baseline: Foo-rt from v1")
	seen := f.seen()
	if seen["docs"] != "none" {
		t.Errorf("docs got a baseline: %q", seen["docs"])
	}
	for _, p := range []string{"rt", "ux", "wb"} {
		if !strings.HasPrefix(seen["Foo-"+p], "v1 /") {
			t.Errorf("Foo-%s: the report saw %q, want its v1 baseline", p, seen["Foo-"+p])
		}
	}
	if strings.Contains(f.out, "schema-risk baseline: docs") {
		t.Errorf("docs was extracted: %s", f.out)
	}
	f.noBaselineLeft("docs")
}

// TestIncompleteCrossRefIgnoresDocs: a docs/ directory without a .deploy-baseline is not what makes a run
// INCOMPLETE, so the schema-risk cross-reference is not printed for it alone.
func TestIncompleteCrossRefIgnoresDocs(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.write(f.root+"/Foo/docs/README.md", "plain notes")
	for _, p := range []string{"rt", "ux", "wb"} {
		f.mkdir(f.artifact(p) + "/.deploy-baseline")
	}
	f.does(reportScript, "exit 4")
	f.wants("docs cross-ref", f.build(), ExitReportPartial, msgReportIncomplete)
	if strings.Contains(f.err, msgSchemaRiskCrossRef) {
		t.Errorf("the schema-risk cross-reference was printed for docs/: %s", f.err)
	}
}
