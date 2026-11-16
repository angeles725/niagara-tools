package writeops

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
)

// A journaled unlinked input with two bad slot names is refused for the first field checked,
// source_slot, as before escaped target slots were accepted (verifier recheck 2026-11-15).
func TestUnlinkedSpecRefusesTheFirstBadFieldFirst(t *testing.T) {
	_, err := unlinkedSpec(obj("source_slot", "a b", "target_path", "", "target_slot", "c d"))
	want := "source_slot must be a plain slot name (letters, digits, underscore), got 'a b'"
	if err == nil || plan.PyStr(err) != want {
		t.Fatalf("err = %v, want %s", err, want)
	}
	if _, err := unlinkedSpec(obj("source_slot", "out", "target_path", "", "target_slot", "c d")); err == nil ||
		!strings.Contains(plan.PyStr(err), "'c d'") {
		t.Fatalf("bad target_slot: %v", err)
	}
}
