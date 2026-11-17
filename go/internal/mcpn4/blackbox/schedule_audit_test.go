package blackbox

// n4_schedule_audit (station-audits S12): one read-only call lists every schedule with its out,
// next event, the special events inside the horizon, and the schedules nothing reads, from the
// six schedules recorded live (testdata/schedules.json) on a fake station.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// saList is the BQL answer listing the schedules with their knob counts: BoolSch and EnumSch
// are read by links, NumSch and StrSch are not (the calendar is referenced, never linked).
const saList = "Slot Path,Knob Count\nslot:/Folder/BoolSch,1\nslot:/Folder/NumSch,0\nslot:/Folder/EnumSch,2\n" +
	"slot:/Folder/StrSch,0\nslot:/Folder/Cal,0\n"

// saRig is a servlet rig holding the recorded schedules (StrSch without nextTime, and its type
// no longer describing it, so its next time cannot be read), a station dated 2026-11-20 and the
// list answered from routes. ref, when not "", repoints BoolSch's reference event.
func saRig(t *testing.T, ref string, routes ...boxstation.OrdRoute) (*Rig, *boxstation.Servlets) {
	t.Helper()
	f := loadScheduleFixture(t)
	r, sv := servletRig(t, nil)
	str := f.Types["schedule:StringSchedule"].(map[string]any)
	kept := []any{}
	for _, c := range str["c"].([]any) {
		if c.(map[string]any)["n"] != "nextTime" {
			kept = append(kept, c)
		}
	}
	str["c"] = kept
	r.Station.SetRegistry(f.Types)
	for _, name := range []string{"BoolSch", "NumSch", "EnumSch", "StrSch", "Cal"} {
		body := f.Schedules[name]
		if name == "StrSch" {
			slots := []any{}
			for _, s := range body["s"].([]any) {
				if s.(map[string]any)["n"] != "nextTime" {
					slots = append(slots, s)
				}
			}
			body["s"] = slots
		}
		install(r, r.Add("3", name, body["t"].(string)).Handle, body)
	}
	if ref != "" {
		r.Set(handleAt(r, "BoolSch", "schedule/specialEvents/Holidays/days"), "ref",
			map[string]any{"nm": "p", "t": "baja:Ord", "v": ref})
	}
	shxSystemService(r.Station, map[string][2]string{"systemTime": {"baja:AbsTime", "2026-11-20T10:00:00.000-06:00"}})
	if len(routes) == 0 {
		routes = []boxstation.OrdRoute{{Needle: "knobCount", Body: []byte(saList)}}
	}
	routeOrds(sv, routes...)
	return r, sv
}

// saEntry is the schedules entry of the Folder child name.
func saEntry(t *testing.T, out map[string]any, name string) map[string]any {
	t.Helper()
	for _, e := range objects(out["schedules"]) {
		if e["schedule"] == "station:|slot:/Folder/"+name {
			return e
		}
	}
	t.Fatalf("no entry for %s in %s", name, jsonOf(t, out["schedules"]))
	return nil
}

func TestScheduleAuditReportsALinkedWeeklyScheduleWithItsNextEvent(t *testing.T) {
	r, _ := saRig(t, "")
	out := r.Srv.OK("n4_schedule_audit", Args{})
	e := saEntry(t, out, "BoolSch")
	want := `{"type":"schedule:BooleanSchedule","kind":"boolean","out_value":true,` +
		`"next_time":"2026-11-16T00:00:00.000-06:00","next_value":false,"linked_targets":1}`
	got := jsonOf(t, map[string]any{"type": e["type"], "kind": e["kind"], "out_value": e["out_value"],
		"next_time": e["next_time"], "next_value": e["next_value"], "linked_targets": e["linked_targets"]})
	if got != canon(t, want) {
		t.Errorf("BoolSch =\n%s\nwant\n%s", got, want)
	}
	if out["today"] != "2026-11-20" || out["horizon_days"] != float64(7) || out["truncated"] != false {
		t.Errorf("today %v horizon %v truncated %v", out["today"], out["horizon_days"], out["truncated"])
	}
}

func TestScheduleAuditListsTheSchedulesNothingReads(t *testing.T) {
	r, _ := saRig(t, "")
	out := r.Srv.OK("n4_schedule_audit", Args{})
	want := `[{"schedule":"station:|slot:/Folder/NumSch"},{"schedule":"station:|slot:/Folder/StrSch"}]`
	if got := jsonOf(t, out["unlinked"]); got != canon(t, want) {
		t.Errorf("unlinked = %s, want %s (a calendar is never listed)", got, want)
	}
	if cal := saEntry(t, out, "Cal"); cal["referenced_by"] != float64(0) || cal["linked_targets"] != float64(0) {
		t.Errorf("Cal referenced_by %v linked %v", cal["referenced_by"], cal["linked_targets"])
	}
	counts := out["counts"].(map[string]any)
	if counts["schedules"] != float64(5) || counts["calendars"] != float64(1) || counts["unlinked"] != float64(2) {
		t.Errorf("counts %v", counts)
	}
}

func TestScheduleAuditPlacesOnlyTheSpecialEventsInsideTheHorizon(t *testing.T) {
	r, _ := saRig(t, "")
	out := r.Srv.OK("n4_schedule_audit", Args{"horizon": "7d"})
	ev := saEntry(t, out, "BoolSch")["special_events"].(map[string]any)
	want := `[{"name":"Thanksgiving","kind":"week_and_day","date":"2026-11-26",` +
		`"rule":{"month":"nov","week":"week4","weekday":"thursday"},"ranges":[{"start":"06:00","end":"12:30","value":true}]}]`
	if ev["count"] != float64(5) || jsonOf(t, ev["upcoming"]) != canon(t, want) {
		t.Errorf("special_events count %v upcoming\n%s\nwant (Christmas Break on 12-24 is outside)\n%s",
			ev["count"], jsonOf(t, ev["upcoming"]), want)
	}
	cals := jsonOf(t, saEntry(t, out, "BoolSch")["calendars"])
	if !strings.Contains(cals, `"ord":"station:|slot:/ImplSCH/Cal","resolves":false`) {
		t.Errorf("an unresolved reference = %s", cals)
	}
}

func TestScheduleAuditFollowsAReferenceToItsCalendar(t *testing.T) {
	r, _ := saRig(t, "station:|slot:/Folder/Cal")
	out := r.Srv.OK("n4_schedule_audit", Args{"horizon": "45d"})
	b := saEntry(t, out, "BoolSch")
	if got := jsonOf(t, b["calendars"]); got != canon(t, `[{"ord":"station:|slot:/Folder/Cal","resolves":true}]`) {
		t.Errorf("calendars = %s", got)
	}
	found := false
	for _, u := range objects(b["special_events"].(map[string]any)["upcoming"]) {
		if u["name"] == "Holidays" {
			found = u["date"] == "2026-12-28" && u["calendar"] == "station:|slot:/Folder/Cal"
		}
	}
	if !found {
		t.Errorf("Holidays not placed on the calendar's Shutdown: %s", jsonOf(t, b["special_events"]))
	}
	if saEntry(t, out, "Cal")["referenced_by"] != float64(1) {
		t.Errorf("Cal referenced_by = %v", saEntry(t, out, "Cal")["referenced_by"])
	}
}

func TestScheduleAuditLeavesAnUnreadableNextTimeNullWithANote(t *testing.T) {
	r, _ := saRig(t, "")
	out := r.Srv.OK("n4_schedule_audit", Args{})
	e := saEntry(t, out, "StrSch")
	notes := jsonOf(t, e["notes"])
	if e["next_time"] != nil || !strings.Contains(notes, "next_time not read") || !strings.Contains(notes, "nextTime") {
		t.Errorf("StrSch next_time %v notes %s", e["next_time"], notes)
	}
	if strings.Contains(jsonOf(t, out["no_next_event"]), "StrSch") {
		t.Errorf("an unreadable next time is not 'no next event': %s", jsonOf(t, out["no_next_event"]))
	}
}

func TestScheduleAuditWithoutKnobCountLeavesLinksUnknown(t *testing.T) {
	list := "Slot Path\nslot:/Folder/NumSch\n"
	r, _ := saRig(t, "", boxstation.OrdRoute{Needle: "select slotPath from schedule:ControlSchedule", Body: []byte(list)})
	out := r.Srv.OK("n4_schedule_audit", Args{})
	if out["unlinked"] != nil || saEntry(t, out, "NumSch")["linked_targets"] != nil ||
		!strings.Contains(jsonOf(t, out["notes"]), "knobCount") {
		t.Errorf("unlinked %v notes %v", out["unlinked"], out["notes"])
	}
}

func TestScheduleAuditRefusesABadHorizon(t *testing.T) {
	r, _ := saRig(t, "")
	for _, h := range []string{"0d", "1w", "400d", "7"} {
		if msg := r.Srv.Err("n4_schedule_audit", Args{"horizon": h}); !strings.Contains(msg, "horizon must be") {
			t.Errorf("horizon %s: %s", h, msg)
		}
	}
}
