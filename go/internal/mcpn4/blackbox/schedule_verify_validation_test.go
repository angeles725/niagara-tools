package blackbox

// Independent probes of n4_set_schedule (weekly + default output), validation and scope: every
// fault named before any write is sent, value types per kind, and the write scope.

import (
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func countSyncs(r *Rig) *int64 {
	var n int64
	r.Station.SetFaults(boxstation.Faults{Refuse: func(_ *boxstation.Station, sck string, _ map[string]any) error {
		if sck == "syncTo" {
			atomic.AddInt64(&n, 1)
		}
		return nil
	}})
	return &n
}

func faultsOf(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, "- ") {
			out = append(out, strings.TrimPrefix(l, "- "))
		}
	}
	sort.Strings(out)
	return out
}

func TestVerifySetScheduleListsEveryFaultOnceAndSendsNoWrite(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	journal := len(r.Lines("journal.jsonl"))
	weekly := map[string]any{
		"Monday": []any{},
		"monday": []any{
			map[string]any{"start": "8:00", "end": "12:00", "value": true},
			map[string]any{"start": "24:00", "end": "24:00", "value": true},
			map[string]any{"start": "10:00", "end": "10:00", "value": true},
			map[string]any{"start": "11:00", "end": "09:00", "value": true},
			map[string]any{"start": "13:00", "end": "15:00", "value": nil},
			map[string]any{"start": "14:00", "end": "16:00"},
			map[string]any{"start": "16:00", "end": "00:00", "value": true},
			map[string]any{"start": "17:00", "end": "18:60", "value": true},
		},
		"tuesday":   "08:00-12:00",
		"wednesday": []any{5},
	}
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": weekly, "default_output": "on"}))
	want := []string{
		"weekly: unknown day 'Monday' (use monday, tuesday, wednesday, thursday, friday, saturday, sunday)",
		"weekly.monday[0]: start '8:00' is not a time HH:MM from 00:00 to 23:59",
		"weekly.monday[1]: start '24:00' is not a time HH:MM from 00:00 to 23:59",
		"weekly.monday[2]: end '10:00' is not after start '10:00'",
		"weekly.monday[3]: end '09:00' is not after start '11:00'",
		"weekly.monday[4]: value null is not allowed for a range (only a default output may be null)",
		"weekly.monday[5]: value is missing",
		"weekly.monday[6]: end '00:00' is not after start '16:00'",
		"weekly.monday[7]: end '18:60' is not a time HH:MM from 00:01 to 24:00",
		"weekly.tuesday: must be a list of {start, end, value}",
		"weekly.wednesday[0]: must be an object {start, end, value}",
		"default_output: value 'on' is not a boolean (a boolean schedule takes true or false)",
	}
	sort.Strings(want)
	if got := faultsOf(text); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("faults:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !strings.HasPrefix(text, fmt.Sprintf("%d fault(s) in the request, nothing was sent:", len(want))) {
		t.Errorf("header: %q", text)
	}
	if n := atomic.LoadInt64(syncs); n != 0 {
		t.Errorf("%d syncTo frames were sent", n)
	}
	if n := len(r.Lines("journal.jsonl")); n != journal {
		t.Errorf("journal grew from %d to %d lines", journal, n)
	}
}

func TestVerifySetScheduleRefusesOverlapsAndAcceptsTouchingRanges(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	text := r.Srv.Err("n4_set_schedule", setSchedule("NumSch", Args{"weekly": map[string]any{"monday": []any{
		map[string]any{"start": "08:00", "end": "18:00", "value": 1},
		map[string]any{"start": "09:00", "end": "10:00", "value": 2},
		map[string]any{"start": "11:00", "end": "12:00", "value": 3}}}}))
	for _, w := range []string{"2 fault(s)", "weekly.monday: ranges 08:00-18:00 and 09:00-10:00 overlap",
		"weekly.monday: ranges 08:00-18:00 and 11:00-12:00 overlap"} {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in %q", w, text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("NumSch", Args{"weekly": map[string]any{"monday": []any{
		map[string]any{"start": "12:00", "end": "24:00", "value": 2},
		map[string]any{"start": "00:00", "end": "12:00", "value": 1}}}}))
	if out["verdict"] != "verified" {
		t.Errorf("touching ranges: %v", out)
	}
	if got := jsonOf(t, readSchedule(r, "NumSch")["weekly"].(map[string]any)["monday"]); got != canon(t,
		`[{"start":"00:00","end":"12:00","value":1.0},{"start":"12:00","end":"24:00","value":2.0}]`) {
		t.Errorf("monday = %s", got)
	}
}

func TestVerifySetScheduleChecksTheValueTypeOfEachKind(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	day := func(v any) Args {
		return Args{"weekly": map[string]any{"friday": []any{map[string]any{"start": "01:00", "end": "02:00", "value": v}}}}
	}
	cases := []struct {
		name  string
		value any
		want  string
	}{
		{"BoolSch", 1, "value 1 is not a boolean (a boolean schedule takes true or false)"},
		{"BoolSch", "true", "value 'true' is not a boolean"},
		{"NumSch", true, "value True is not a number (a numeric schedule takes a number)"},
		{"NumSch", "19.5", "value '19.5' is not a number"},
		{"EnumSch", 2.5, "value 2.5 is not an enum ordinal (an enum schedule takes a whole number >= 0)"},
		{"EnumSch", -1, "value -1 is not an enum ordinal"},
		{"EnumSch", "on", "value 'on' is not an enum ordinal"},
		{"StrSch", 5, "value 5 is not a string (a string schedule takes a string)"},
		{"StrSch", false, "value False is not a string"},
	}
	for _, c := range cases {
		text := r.Srv.Err("n4_set_schedule", setSchedule(c.name, day(c.value)))
		if !strings.Contains(text, "1 fault(s)") || !strings.Contains(text, "weekly.friday[0]: "+c.want) {
			t.Errorf("%s %v: %q", c.name, c.value, text)
		}
		text = r.Srv.Err("n4_set_schedule", setSchedule(c.name, Args{"default_output": c.value}))
		if !strings.Contains(text, "default_output: "+c.want) {
			t.Errorf("%s default %v: %q", c.name, c.value, text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

func TestVerifySetScheduleRefusesAWeeklyThatIsNotAnObject(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	if code := r.Srv.CallError("n4_set_schedule", Args{"ord": Folder + "/BoolSch", "weekly": []any{}}); code != -32602 {
		t.Errorf("code = %d", code)
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

func TestVerifySetScheduleStaysInsideTheWriteScope(t *testing.T) {
	r := scheduleWriteRig(t)
	dry := r.Srv.Dry("n4_set_schedule", setSchedule("BoolSch", Args{"default_output": true}))
	for _, scope := range []string{Folder + "/NumSch", Folder + "/Bool"} {
		r.Restart(func(o *Options) { o.WriteScopes = []string{scope} })
		r.ConnectVerified()
		syncs := countSyncs(r)
		args := setSchedule("BoolSch", Args{"default_output": true})
		if text := r.Srv.Err("n4_set_schedule", args); !strings.Contains(text, "outside") {
			t.Errorf("scope %s, dry run: %q", scope, text)
		}
		if text := r.Srv.Err("n4_set_schedule", Execute(args, dry["confirmation_token"])); !strings.Contains(text, "outside") {
			t.Errorf("scope %s, wider token: %q", scope, text)
		}
		if atomic.LoadInt64(syncs) != 0 {
			t.Errorf("scope %s: a write was sent", scope)
		}
	}
}

func TestVerifySetScheduleInsideANestedScope(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Restart(func(o *Options) { o.WriteScopes = []string{Folder + "/NumSch"} })
	r.ConnectVerified()
	if out := r.Srv.RunWrite("n4_set_schedule", setSchedule("NumSch", Args{"default_output": 4})); out["verdict"] != "verified" {
		t.Errorf("out = %v", out)
	}
}

func TestVerifySetScheduleChecksAnEnumValueAgainstTheRangeFacet(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Set(enumHandle(r), "facets",
		map[string]any{"nm": "p", "t": "baja:Facets", "v": "range=E:{Off=0,On=1,Auto=3}"})
	syncs := countSyncs(r)
	text := r.Srv.Err("n4_set_schedule", setSchedule("EnumSch", Args{"default_output": 2,
		"weekly": map[string]any{"monday": []any{map[string]any{"start": "01:00", "end": "02:00", "value": 4}}}}))
	for _, w := range []string{"2 fault(s)", "default_output: value 2 is not an ordinal of the schedule's range facet",
		"weekly.monday[0]: value 4 is not an ordinal of the schedule's range facet"} {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in %q", w, text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
	r.Station.SetFaults(boxstation.Faults{})
	if out := r.Srv.RunWrite("n4_set_schedule", setSchedule("EnumSch", Args{"default_output": 3})); out["verdict"] != "verified" {
		t.Errorf("ordinal 3 of the range: %v", out)
	}
}

func enumHandle(r *Rig) (h string) {
	r.Station.Do(func(st *boxstation.Station) { h = st.ByHandle("3").Child("EnumSch").Handle })
	return h
}
