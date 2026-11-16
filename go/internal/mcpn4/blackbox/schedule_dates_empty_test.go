package blackbox

import (
	"strings"
	"testing"
)

// TestSetScheduleEmptyDateListsAreNothingToSet: add_date: [] and remove_date: [] on a calendar or
// a trigger schedule change nothing, so the call is refused as "nothing to set" instead of
// planning an empty date part (review R3-001: that part used to crash the planner).
func TestSetScheduleEmptyDateListsAreNothingToSet(t *testing.T) {
	r := scheduleWriteRig(t)
	for _, name := range []string{"Cal", "TrigSch"} {
		before := readSchedule(r, name)
		text := r.Srv.Err("n4_set_schedule", setSchedule(name, Args{"add_date": []any{}, "remove_date": []any{}}))
		if !strings.Contains(text, "nothing to set") {
			t.Errorf("%s: %q", name, text)
		}
		if got, want := jsonOf(t, readSchedule(r, name)["dates"]), jsonOf(t, before["dates"]); got != want {
			t.Errorf("%s changed: %s", name, got)
		}
	}
}
