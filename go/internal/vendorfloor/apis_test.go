package vendorfloor

import (
	"strings"
	"testing"
)

// The shipped table must be valid: without this, a shared name added to Default is only noticed at run time.
func TestDefaultTableIsDisjoint(t *testing.T) {
	t.Parallel()
	if err := Default.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateNamesASharedMethod(t *testing.T) {
	t.Parallel()
	apis := APIs{Member: map[string]int{"at": 92}, Qualified: map[string]int{"Array.at": 92, "Object.hasOwn": 93}}
	err := apis.Validate()
	if err == nil || err.Error() != "MEMBER and QUALIFIED share the method name 'at' (Array.at)" {
		t.Errorf("Validate() = %v", err)
	}
}

func TestNamesCoverTheTable(t *testing.T) {
	t.Parallel()
	got := strings.Join(Default.Names(), " ")
	want := "FinalizationRegistry WeakRef any at findLast findLastIndex groupBy hasOwn replaceAll structuredClone timeout toReversed toSorted toSpliced"
	if got != want {
		t.Errorf("Names() = %s\nwant      %s", got, want)
	}
}
