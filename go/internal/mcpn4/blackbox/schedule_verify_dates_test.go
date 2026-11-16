package blackbox

// Independent probes of n4_set_schedule calendar dates and trigger times (slice 4): each date
// kind reads back, the rollback restores the station tree exactly, faults are named before any
// write, the write scope holds, and the rollback guard and read-back see dates and times.

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func date(name, kind string, rule map[string]any) map[string]any {
	return map[string]any{"name": name, "kind": kind, "rule": rule}
}

var everyDateKind = []any{
	date("May Day", "date", map[string]any{"month": "may", "day": 1}),
	date("Span", "date_range", map[string]any{"start": map[string]any{"year": 2099, "month": "jan", "day": 1},
		"end": map[string]any{"year": 2099, "month": "feb", "day": 2}}),
	date("Last Mondays", "week_and_day", map[string]any{"week": "last7days", "weekday": "monday"}),
	date("Custom", "custom", map[string]any{"months": []any{"even"}, "days": []any{15}}),
}

func TestVerifyEveryDateKindReadsBackAndRollsBackTheTreeExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	for _, name := range []string{"Cal", "TrigSch"} {
		before := treeOf(r, name, false)
		out := r.Srv.RunWrite("n4_set_schedule", setSchedule(name, Args{"add_date": everyDateKind}))
		if out["verdict"] != "verified" {
			t.Fatalf("%s: %v", name, out)
		}
		dates := jsonOf(t, readSchedule(r, name)["dates"])
		for _, w := range []string{`"name":"May Day","rule":{"day":1,"month":"may","weekday":"*","year":"*"}`,
			`"name":"Last Mondays","rule":{"month":"*","week":"last7days","weekday":"monday"}`,
			`"name":"Custom","rule":{"days":[15],"months":["even"],"weekdays":[],"weeks":[],"year":"*"}`} {
			if !strings.Contains(dates, w) {
				t.Errorf("%s: missing %s in %s", name, w, dates)
			}
		}
		r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]})
		if after := treeOf(r, name, false); after != before {
			t.Errorf("%s tree after add+rollback:\n%s\n---\n%s", name, before, after)
		}
	}
}

func TestVerifyARemovedDateComesBackWithItsTreeExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	before := treeOf(r, "Cal", false)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("Cal", Args{"remove_date": []any{"NewYear", "Shutdown"}}))
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
		t.Errorf("rollback %v", back)
	}
	if after := treeOf(r, "Cal", false); after != before {
		t.Errorf("tree:\n%s\n---\n%s", before, after)
	}
}

func TestVerifyDateAndTimeFaultsAreNamedAndNothingIsSent(t *testing.T) {
	r := scheduleWriteRig(t)
	syncs := countSyncs(r)
	text := r.Srv.Err("n4_set_schedule", setSchedule("TrigSch", Args{"times": []any{"06:00", "6:30", "06:00", "24:00", 7},
		"add_date":    []any{date("", "date", map[string]any{"day": 0}), date("X", "date", map[string]any{"year": 2020})},
		"remove_date": []any{"Nope"}}))
	for _, w := range []string{"times[1]: '6:30' is not a time", "times[2]: '06:00' is given twice",
		"times[3]: '24:00' is not a time", "times[4]: 7 is not a time", "add_date[0].name: must be a non-empty string",
		"add_date[0].rule.day: 0 is not a day of the month", "add_date[1]: 'X' can never occur again",
		"remove_date: no date is named 'Nope'"} {
		if !strings.Contains(text, w) {
			t.Errorf("missing %q in %q", w, text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

func TestVerifyTimesReplaceAllAndRollBackExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	before := treeOf(r, "TrigSch", false)
	for _, times := range [][]any{{"23:59", "00:00"}, {}} {
		out := r.Srv.RunWrite("n4_set_schedule", setSchedule("TrigSch", Args{"times": times}))
		got := jsonOf(t, readSchedule(r, "TrigSch")["times"])
		if want := map[int]string{2: `["00:00","23:59"]`, 0: `[]`}[len(times)]; out["verdict"] != "verified" || got != want {
			t.Errorf("times %v: verdict %v, read %s", times, out["verdict"], got)
		}
		r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]})
		if after := treeOf(r, "TrigSch", false); after != before {
			t.Errorf("tree after %v + rollback:\n%s\n---\n%s", times, before, after)
		}
	}
}

func TestVerifyDatesStayInsideTheWriteScope(t *testing.T) {
	r := scheduleWriteRig(t)
	args := setSchedule("Cal", Args{"remove_date": []any{"NewYear"}})
	dry := r.Srv.Dry("n4_set_schedule", args)
	r.Restart(func(o *Options) { o.WriteScopes = []string{Folder + "/Ca"} })
	r.ConnectVerified()
	syncs := countSyncs(r)
	for _, a := range []Args{args, Execute(args, dry["confirmation_token"])} {
		if text := r.Srv.Err("n4_set_schedule", a); !strings.Contains(text, "outside") {
			t.Errorf("%q", text)
		}
	}
	if atomic.LoadInt64(syncs) != 0 {
		t.Errorf("a write was sent")
	}
}

func TestVerifyTimesRollbackDoesNotOverwriteTimesChangedSince(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("TrigSch", Args{"times": []any{"05:00"}}))
	r.Station.Do(func(st *boxstation.Station) {
		st.Root().Child("Folder").Child("TrigSch").Child("times").Children[0].Child("start").Value = "05:30:00.000"
	})
	if res := r.Srv.Call("n4_rollback", Args{"batch_id": out["batch_id"]}); !res.IsError || !strings.Contains(res.Text, "changed since batch") {
		t.Errorf("the rollback plans to overwrite times edited since the batch: %.200s", res.Text)
	}
}

func TestVerifyDatesRollbackReportsADateTheStationDidNotRestore(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("Cal", Args{"remove_date": []any{"NewYear"}}))
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		if d := st.Root().Child("Folder").Child("Cal").Child("NewYear"); d != nil && d.Child("monthSchedule") != nil {
			if s := d.Child("monthSchedule").Child("set"); s != nil {
				s.Value = "6"
			}
		}
	}})
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] == "verified" {
		t.Errorf("rollback verified though NewYear came back in july: %v", back["observed"])
	}
}
