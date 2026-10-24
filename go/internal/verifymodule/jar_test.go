package verifymodule

import "testing"

// TestTextAbsentEntry: an entry the jar does not hold reads as "" (unzip -p … || true), never a nil dereference.
func TestTextAbsentEntry(t *testing.T) {
	j := &jarFile{}
	if got := j.text("module.palette"); got != "" {
		t.Errorf(`text(absent) = %q, want ""`, got)
	}
}
