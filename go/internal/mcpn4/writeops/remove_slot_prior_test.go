package writeops_test

import (
	"strings"
	"testing"
)

// A removed slot written by a link records its last value (prior) on the unlinked input, as a
// component's linked input does, so n4_rollback reports it (verifier recheck 2026-11-15).
func TestRemoveSimpleSlotLinkedIntoRecordsItsPriorValue(t *testing.T) {
	p, err := esPlan(t, "dyn")
	if err != nil {
		t.Fatal(err)
	}
	inv := esJSON(t, p.Inverse)
	if !strings.Contains(inv, `"target_slot": "dyn", "prior": {"nm": "p", "t": "baja:Double", "v": "3.0"}`) {
		t.Errorf("no prior on the unlinked input: %s", inv)
	}
	if notes := esJSON(t, p.Notes); !strings.Contains(notes, "its last value is recorded and reported by n4_rollback") {
		t.Errorf("notes %s", notes)
	}
}
