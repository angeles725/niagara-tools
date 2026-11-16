package blackbox

// Independent probes of n4_set_schedule special events (slice 4): every event kind encodes and
// reads back, the order is the priority, the expiry rule, every fault named before any write,
// and the rollback guard and read-back of the specialEvents set.

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func allDay(v any) []any { return []any{map[string]any{"start": "00:00", "end": "24:00", "value": v}} }

func event(name, kind string, rule map[string]any, v any) map[string]any {
	return map[string]any{"name": name, "kind": kind, "rule": rule, "ranges": allDay(v)}
}

func TestVerifyEveryEventKindEncodesAndReadsBackLastInOrder(t *testing.T) {
	r := scheduleWriteRig(t)
	events := []any{
		event("D", "date", map[string]any{"year": 2099, "month": "may", "day": 1, "weekday": "friday"}, 1.5),
		event("R", "date_range", map[string]any{"start": map[string]any{"month": "jun", "day": 1},
			"end": map[string]any{"year": 2099, "month": "jun", "day": 30}}, 2),
		event("W", "week_and_day", map[string]any{"month": "odd", "week": "last7days", "weekday": "monday"}, 3),
		event("C", "custom", map[string]any{"year": 2099, "months": []any{"jan", "feb"}, "days": []any{1, 31},
			"weeks": []any{"week2"}, "weekdays": []any{"sunday", "saturday"}}, 4),
		event("Ref 1", "reference", map[string]any{"calendar": Folder + "/Cal"}, 5),
	}
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("NumSch", Args{"add_special_event": events}))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	got := objects(readSchedule(r, "NumSch")["special_events"])
	n := len(got)
	want := []string{
		`{"name":"D","kind":"date","rule":{"year":2099,"month":"may","day":1,"weekday":"friday"}}`,
		`{"name":"R","kind":"date_range","rule":{"start":{"year":"*","month":"jun","day":1,"weekday":"*"},"end":{"year":2099,"month":"jun","day":30,"weekday":"*"}}}`,
		`{"name":"W","kind":"week_and_day","rule":{"month":"odd","week":"last7days","weekday":"monday"}}`,
		`{"name":"C","kind":"custom","rule":{"year":2099,"months":["jan","feb"],"days":[1,31],"weeks":["week2"],"weekdays":["sunday","saturday"]}}`,
		`{"name":"Ref 1","kind":"reference","rule":{"calendar":"` + Folder + `/Cal"}}`,
	}
	for i, w := range want {
		e := got[n-5+i]
		if s := jsonOf(t, map[string]any{"name": e["name"], "kind": e["kind"], "rule": e["rule"]}); s != canon(t, w) {
			t.Errorf("event %d = %s\nwant %s", i, s, canon(t, w))
		}
		if e["priority"] != float64(n-4+i) {
			t.Errorf("event %d priority %v of %d", i, e["priority"], n)
		}
		if s := jsonOf(t, e["ranges"]); !strings.Contains(s, `"end":"24:00","start":"00:00"`) {
			t.Errorf("event %d ranges %s", i, s)
		}
	}
}

func TestVerifyExpiryIsJudgedAgainstToday(t *testing.T) {
	r := scheduleWriteRig(t)
	now := time.Now()
	ymd := func(d time.Time) map[string]any {
		return map[string]any{"year": d.Year(), "month": strings.ToLower(d.Month().String()[:3]), "day": d.Day()}
	}
	yesterday, tomorrow := ymd(now.AddDate(0, 0, -1)), ymd(now.AddDate(0, 0, 1))
	expired := []any{event("Y", "date", yesterday, true),
		event("YR", "date_range", map[string]any{"start": ymd(now.AddDate(0, 0, -9)), "end": yesterday}, true)}
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": expired}))
	for _, w := range []string{"2 fault(s)", "add_special_event[0]: 'Y' can never occur again", "add_special_event[1]: 'YR' can never occur again"} {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in %q", w, text)
		}
	}
	live := []any{event("T", "date", ymd(now), true), event("N", "date", tomorrow, true),
		event("Any", "date", map[string]any{"month": "jan", "day": 1}, true),
		event("Open", "date_range", map[string]any{"start": yesterday, "end": tomorrow}, true)}
	if out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": live})); out["verdict"] != "verified" {
		t.Errorf("live events: %v", out)
	}
	allowed := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": expired, "allow_expired": true}))
	if allowed["verdict"] != "verified" {
		t.Errorf("allow_expired: %v", allowed)
	}
}

func TestVerifyEventFaultsAreAllNamedAndNothingIsSent(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	bad := []any{
		"x",
		map[string]any{"name": "", "kind": "date", "rule": map[string]any{"month": "jan", "day": 32, "colour": "red"}, "ranges": []any{}},
		map[string]any{"name": "K", "kind": "weekly", "rule": map[string]any{}, "ranges": []any{}},
		map[string]any{"name": "Wk", "kind": "week_and_day", "rule": map[string]any{"week": "week9", "weekday": "funday"}, "ranges": []any{}},
		map[string]any{"name": "Yr", "kind": "custom", "rule": map[string]any{"year": 1900}},
		map[string]any{"name": "Rg", "kind": "date", "rule": map[string]any{}, "ranges": []any{
			map[string]any{"start": "01:00", "end": "03:00", "value": 1}, map[string]any{"start": "02:00", "end": "04:00", "value": false}}},
		map[string]any{"name": "Rf", "kind": "reference", "rule": map[string]any{"calendar": "Cal"}, "ranges": []any{}},
	}
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": bad, "remove_special_event": []any{"Nope"}}))
	want := []string{
		"add_special_event[0]: must be an object {name, kind, rule, ranges}",
		"add_special_event[1].name: must be a non-empty string",
		"add_special_event[1].rule: unknown field 'colour'",
		"add_special_event[1].rule.day: 32 is not a day of the month (1-31)",
		"add_special_event[2].rule: unknown kind 'weekly'",
		"add_special_event[3].rule.week: 'week9' is not a week",
		"add_special_event[3].rule.weekday: 'funday' is not a weekday",
		"add_special_event[4].rule.year: 1900 is not a year (1970-9999)",
		"add_special_event[4].ranges: must be a list of {start, end, value}",
		"add_special_event[5].ranges[0]: value 1 is not a boolean",
		"add_special_event[5].ranges: ranges 01:00-03:00 and 02:00-04:00 overlap",
		"add_special_event[6].rule.calendar: 'Cal' is not a calendar's slot ORD",
		"remove_special_event: no event is named 'Nope'",
	}
	for _, w := range want {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q", w)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

func TestVerifyARemoveNamedTwiceIsAFaultOrOneRemoval(t *testing.T) {
	r := scheduleWriteRig(t)
	res := r.Srv.Call("n4_set_schedule", setSchedule("BoolSch", Args{"remove_special_event": []any{"Summer", "Summer"}}))
	if !res.IsError && strings.Contains(res.Text, "2 removed") {
		t.Errorf("one event removed, the plan says 2: %.400s", res.Text)
	}
}

// The specialEvents list edited after the batch: the rollback must not overwrite it.
func TestVerifyEventsRollbackDoesNotOverwriteAListChangedSince(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{
		event("Later", "date", map[string]any{"year": 2099, "month": "jan", "day": 2}, true)}}))
	r.Station.Do(func(st *boxstation.Station) {
		se := st.Root().Child("Folder").Child("BoolSch").Child("schedule").Child("specialEvents")
		se.Children = se.Children[1:] // an operator deletes the first event
	})
	if res := r.Srv.Call("n4_rollback", Args{"batch_id": out["batch_id"]}); !res.IsError || !strings.Contains(res.Text, "changed since batch") {
		t.Errorf("the rollback plans to overwrite an edited event list: %.200s", res.Text)
	}
}

func TestVerifyEventsRollbackReportsAListTheStationDidNotRestore(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"remove_special_event": []any{"Summer"}}))
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		se := st.Root().Child("Folder").Child("BoolSch").Child("schedule").Child("specialEvents")
		se.Children = se.Children[:1]
	}})
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] == "verified" {
		t.Errorf("rollback verified though the station kept one event: %v", back["observed"])
	}
}

// The station stores the added event with another rule: the write's read-back must say so.
func TestVerifyEventsReadBackComparesTheRule(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		se := st.Root().Child("Folder").Child("BoolSch").Child("schedule").Child("specialEvents")
		if e := se.Child("Later"); e != nil {
			e.Child("days").Child("monthSchedule").Child("set").Value = "5"
		}
	}})
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{
		event("Later", "date", map[string]any{"year": 2099, "month": "jan", "day": 2}, true)}}))
	if out["verdict"] == "verified" {
		t.Errorf("verified though the station holds june, not january: %s", jsonOf(t, out["observed"]))
	}
}
