package rules

import "testing"

func TestAllUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, sp := range All() {
		id := sp.ID()
		if id == "" || seen[id] || sp.Usage == "" || len(sp.Select.Names) == 0 {
			t.Errorf("bad or duplicate spec %q: %+v", id, sp)
		}
		seen[id] = true
	}
}
