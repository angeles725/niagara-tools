package buildmodule

import "testing"

// TestFeatureDocIsForwarded (#1145): --feature-doc <file> goes to report-module as it was given, after the
// options the chain always forwards; the verify gate does not get it. A missing file is report-module's row
// (one unreadable ADVISORY), never a refusal of build.sh.
func TestFeatureDocIsForwarded(t *testing.T) {
	f := newFixture(t)
	doc := f.dir + "/no such feature doc.md" // not created: report-module reads it
	f.wants("feature doc", f.build("--feature-doc", doc), ExitOK)
	want := "report-module.sh <" + f.root + "/Foo> <--niagara-home> <" + f.home + "> <--feature-doc> <" + doc + ">"
	if got := f.call(reportScript); got != want {
		t.Errorf("report call %q, want %q", got, want)
	}
	if got := f.call(gateScript); got != "verify-module.sh <--src> <"+f.root+"/Foo> <"+f.built("rt")+"> <"+f.built("ux")+">" {
		t.Errorf("the gate call carries the feature doc: %q", got)
	}
}

// TestFeatureDocAbsentIsNotForwarded (#1145): without the flag the report-module call is what it was.
func TestFeatureDocAbsentIsNotForwarded(t *testing.T) {
	f := newFixture(t)
	f.wants("default", f.build(), ExitOK)
	if want := "report-module.sh <" + f.root + "/Foo> <--niagara-home> <" + f.home + ">"; f.call(reportScript) != want {
		t.Errorf("report call %q, want %q", f.call(reportScript), want)
	}
}

// TestFeatureDocNeedsAValue (#1145): --feature-doc as the last flag with no value is the usage error (exit 2).
func TestFeatureDocNeedsAValue(t *testing.T) {
	f := newFixture(t)
	f.wants("no value", f.run("--feature-doc"), ExitUsage)
	if f.call(reportScript) != "" || f.call("gradlew") != "" {
		t.Errorf("the chain ran:\n%s", f.calls())
	}
}
