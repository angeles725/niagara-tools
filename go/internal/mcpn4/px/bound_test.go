package px

import "testing"

// Go twin of test_files.TestPxParser.test_bound_component_strips_known_value_slots_and_slot_hops.
func TestBoundComponentStripsKnownValueSlotsAndSlotHops(t *testing.T) {
	for given, w := range map[string]string{"station:|slot:/A/B/out": "station:|slot:/A/B",
		"station:|slot:/A/B|slot:out": "station:|slot:/A/B", "station:|slot:/A/B": "station:|slot:/A/B",
		"slot:/A/B/in10": "slot:/A/B", "slot:/A/B|slot:out\n": "slot:/A/B", "slot:/A/B|slot:o-t": "slot:/A/B|slot:o-t",
		"slot:out": "slot:out", "x:/out": "x:/out", "/out": "", "slot:/A/in17": "slot:/A/in17"} {
		if got := BoundComponent(given); got != w {
			t.Errorf("BoundComponent(%q) = %q, want %q", given, got, w)
		}
	}
}
