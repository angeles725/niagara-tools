package blackbox

// n4_set_slot batch form: duplicate entries reached through different ORD and slot spellings.

import (
	"strings"
	"testing"
)

// Verifier probe 2 (PR #780): a nested slot path through a child component (A, "B/out") and that
// component's own slot (A/B, "out") are the same slot: refused as a duplicate naming both entries.
func TestANestedPathThroughAChildComponentIsADuplicateOfItsOwnSlot(t *testing.T) {
	r := verifiedRig(t)
	a := r.AddWithOut("A", 1.0)
	b := r.Add(a.Handle, "B", "")
	r.Set(b.Handle, "out", StatusNumeric(3.0, "0"))
	writes := []any{
		map[string]any{"ord": Folder + "/A", "slot": "B/out", "value": 5.0, "value_type": "baja:StatusNumeric"},
		map[string]any{"ord": Folder + "/A/B", "slot": "out", "value": 6.0, "value_type": "baja:StatusNumeric"},
	}
	text := r.Srv.Err("n4_set_slot", Args{"writes": writes})
	if !strings.Contains(text, "twice") || !strings.Contains(text, "writes[0]") || !strings.Contains(text, "writes[1]") {
		t.Errorf("refusal = %s", text)
	}
	if got := nodeValue(r, b.Handle, "out", "value"); got != "3.0" {
		t.Errorf("B.out = %v", got)
	}
}
