package blackbox

import (
	"strings"
	"sync/atomic"
	"testing"
)

// A name given twice to remove is a fault, for events and dates alike, and nothing is sent.
func TestSetScheduleRefusesANameRemovedTwice(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	for sched, args := range map[string]Args{
		"BoolSch": {"remove_special_event": []any{"Summer", "Summer"}},
		"Cal":     {"remove_date": []any{"NewYear", "NewYear"}},
	} {
		text := r.Srv.Err("n4_set_schedule", setSchedule(sched, args))
		if !strings.Contains(text, "'Summer' is named twice") && !strings.Contains(text, "'NewYear' is named twice") {
			t.Errorf("%s: %s", sched, text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

// A reference must name an existing calendar: another schedule or a missing ORD is refused.
func TestSetScheduleRefusesAReferenceToNoCalendar(t *testing.T) {
	r := scheduleWriteRig(t)
	for _, ord := range []string{Folder + "/NumSch", Folder + "/Nope"} {
		text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{refEvent(ord)}}))
		if !strings.Contains(text, "the reference '"+ord+"' names no calendar") {
			t.Errorf("%s: %s", ord, text)
		}
	}
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{refEvent(Folder + "/Cal")}}))
	if out["verdict"] != "verified" {
		t.Errorf("a reference to the calendar: %v", out)
	}
}

// refEvent is a reference special event to ord, on all day.
func refEvent(ord string) map[string]any {
	return map[string]any{"name": "Ref", "kind": "reference", "rule": map[string]any{"calendar": ord},
		"ranges": []any{map[string]any{"start": "00:00", "end": "24:00", "value": true}}}
}
