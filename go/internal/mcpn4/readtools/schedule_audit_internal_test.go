package readtools

import "testing"

func TestSaCountReadsOnlyAWholeKnobCount(t *testing.T) {
	for text, want := range map[string]any{"0": 0, "3": 3, "": nil, "-1": nil, "2.0": nil, "null": nil} {
		if got := saCount(text); got != want {
			t.Errorf("saCount(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestSaKeyMatchesACalendarWhateverItsStationPrefix(t *testing.T) {
	for _, ord := range []string{"station:|slot:/A/Cal", "slot:/A/Cal", "local:|fox:|station:|slot:/A/Cal"} {
		if got := saKey(ord); got != "slot:/A/Cal" {
			t.Errorf("saKey(%q) = %q", ord, got)
		}
	}
}
