package lint

import (
	"reflect"
	"testing"
)

// TestPartOutputs: the shared prune list of lint-permissions-xml, lint-adapter-cfg-default and lint-logger-name.
// A part's own test/ and build/ are never entered; a package named test or build under src/ and a group-level
// build/ or test/ are read.
func TestPartOutputs(t *testing.T) {
	want := []string{"*-rt/test", "*-rt/build", "*-ux/test", "*-ux/build", "*-wb/test", "*-wb/build", "*-se/test", "*-se/build"}
	if got := PartOutputs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("PartOutputs() = %q, want %q", got, want)
	}
	PartOutputs()[0] = "changed"
	if PartOutputs()[0] != want[0] {
		t.Fatal("a caller's change to the returned slice leaked into the next call")
	}
	sel := Selection{Names: []string{"*.java"}, PrunePaths: PartOutputs()}
	for _, tc := range []struct {
		path string
		read bool
	}{
		{"/r/M/M-rt/src/com/acme/A.java", true},
		{"/r/M/M-rt/src/com/acme/test/A.java", true},
		{"/r/M/M-ux/src/com/acme/build/A.java", true},
		{"/r/M/build/generated/A.java", true},
		{"/r/M/test/A.java", true},
		{"/r/M/M-rt/test/A.java", false},
		{"/r/M/M-rt/build/generated/A.java", false},
		{"/r/M/M-ux/build/A.java", false},
		{"/r/M/M-wb/test/A.java", false},
		{"/r/M/M-se/build/A.java", false},
	} {
		if got := sel.keeps("/r/M", tc.path); got != tc.read {
			t.Errorf("keeps(%s) = %v, want %v", tc.path, got, tc.read)
		}
	}
}
