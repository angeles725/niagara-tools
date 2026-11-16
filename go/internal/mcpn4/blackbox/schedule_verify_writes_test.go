package blackbox

// Independent probes of n4_set_schedule (weekly + default output), writes and rollback: only the
// days given replaced, the planned ops and inverses, and a rollback that restores the station tree
// exactly or says it did not.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func treeOf(r *Rig, name string, handles bool) string {
	var lines []string
	var walk func(n *boxstation.Node, path string)
	walk = func(n *boxstation.Node, path string) {
		line := fmt.Sprintf("%s t=%s v=%v f=%s", path, n.Type, n.Value, n.Flags)
		if handles {
			line += " h=" + n.Handle
		}
		lines = append(lines, line)
		for _, c := range n.Children {
			walk(c, path+"/"+c.Name)
		}
	}
	r.Station.Do(func(st *boxstation.Station) { walk(st.ByHandle("3").Child(name), name) })
	return strings.Join(lines, "\n")
}

func linesOutside(tree string, prefixes ...string) string {
	var out []string
	for _, l := range strings.Split(tree, "\n") {
		keep := true
		for _, p := range prefixes {
			if strings.HasPrefix(l, p+"/") || strings.HasPrefix(l, p+" ") {
				keep = false
			}
		}
		if keep {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func TestVerifySetSchedulePlansOneSetPerDayWithTheLoadedDayAsInverse(t *testing.T) {
	r := scheduleWriteRig(t)
	args := setSchedule("BoolSch", Args{"weekly": map[string]any{
		"sunday": []any{}, "monday": []any{map[string]any{"start": "08:00", "end": "12:00", "value": true}}}})
	p := planOf(r.Srv.Dry("n4_set_schedule", args))
	ops, inverse := p["ops"].([]any), p["inverse"].([]any)
	if len(ops) != 2 || len(inverse) != 2 {
		t.Fatalf("ops %v inverse %v", ops, inverse)
	}
	for i, day := range []string{"monday", "sunday"} {
		h := handleAt(r, "BoolSch", "schedule/week/"+day)
		op, inv := ops[i].(map[string]any), inverse[i].(map[string]any)
		if op["nm"] != "s" || op["h"] != h || op["n"] != "day" || inv["nm"] != "s" || inv["h"] != h || inv["n"] != "day" {
			t.Errorf("%s: op %v inverse %v", day, op, inv)
		}
	}
	if got := jsonOf(t, inverse[0].(map[string]any)["b"]); got != canon(t, `{"nm":"p","t":"schedule:DaySchedule","s":[
		{"nm":"p","n":"time","t":"schedule:TimeSchedule","s":[{"nm":"p","n":"start","t":"baja:Time","v":"08:00:00.000"},
		{"nm":"p","n":"finish","t":"baja:Time","v":"18:00:00.000"}]}]}`) {
		t.Errorf("monday inverse = %s", got)
	}
}

func TestVerifySetScheduleLeavesEverythingButTheDaysGivenByteIdentical(t *testing.T) {
	r := scheduleWriteRig(t)
	before := map[string]string{}
	for _, n := range []string{"BoolSch", "NumSch", "EnumSch", "StrSch", "Cal", "TrigSch"} {
		before[n] = treeOf(r, n, true)
	}
	r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": map[string]any{"sunday": []any{},
		"monday": []any{map[string]any{"start": "08:00", "end": "12:00", "value": true}}}}))
	for n, tree := range before {
		after := treeOf(r, n, true)
		if n == "BoolSch" {
			days := []string{"BoolSch/schedule/week/monday/day", "BoolSch/schedule/week/sunday/day"}
			tree, after = linesOutside(tree, days...), linesOutside(after, days...)
		}
		if after != tree {
			t.Errorf("%s changed outside the days given:\n%s\n---\n%s", n, tree, after)
		}
	}
	if got := jsonOf(t, readSchedule(r, "BoolSch")["weekly"].(map[string]any)["sunday"]); got != "[]" {
		t.Errorf("sunday = %s", got)
	}
}

func TestVerifySetScheduleRollbackRestoresTheTreeExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	for name, v := range map[string][2]any{"BoolSch": {true, false}, "NumSch": {12, 7.25}, "EnumSch": {1, 3}, "Cal": {false, nil}} {
		before := treeOf(r, name, false)
		args := Args{"default_output": v[0]}
		if v[1] != nil {
			args["weekly"] = map[string]any{"monday": []any{}, "saturday": []any{
				map[string]any{"start": "01:00", "end": "02:00", "value": v[1]},
				map[string]any{"start": "03:00", "end": "24:00", "value": v[1]}}}
		}
		out := r.Srv.RunWrite("n4_set_schedule", setSchedule(name, args))
		if out["verdict"] != "verified" {
			t.Fatalf("%s: %v", name, out)
		}
		if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
			t.Errorf("%s rollback = %v", name, back)
		}
		if after := treeOf(r, name, false); after != before {
			t.Errorf("%s tree not restored:\n%s\n---\n%s", name, before, after)
		}
	}
}

func TestVerifySetScheduleReportsADefaultOutputTheStationDidNotKeep(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		d := st.Root().Child("Folder").Child("NumSch").Child("defaultOutput")
		d.Child("value").Value = "1.0"
	}})
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("NumSch", Args{"default_output": 5}))
	if out["verdict"] != "mismatch" || jsonOf(t, out["observed"]) != canon(t, `{"default_output":{"value":1.0,"status":"ok"}}`) {
		t.Errorf("out = %v", out)
	}
}

func TestVerifySetScheduleRollbackReportsADayTheStationDidNotRestore(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": map[string]any{"monday": []any{}}}))
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		st.Root().Child("Folder").Child("BoolSch").Child("schedule").Child("week").Child("monday").Child("day").Children = nil
	}})
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]})
	if back["verdict"] == "verified" {
		t.Errorf("rollback verdict verified though monday stayed empty: %v", back)
	}
}

func TestVerifySetScheduleRollbackDoesNotOverwriteADayChangedSince(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": map[string]any{"monday": []any{}}}))
	r.Set(handleAt(r, "BoolSch", "schedule/week/monday"), "day",
		daySchedule(timeRange("time", "06:00:00.000", "07:00:00.000", "baja:StatusBoolean", "true", "")))
	res := r.Srv.Call("n4_rollback", Args{"batch_id": out["batch_id"]})
	if !res.IsError || !strings.Contains(res.Text, "changed since batch") {
		t.Errorf("the rollback dry run plans to overwrite monday, changed since the batch: %.300s", res.Text)
	}
}

func TestVerifySetScheduleRollbackDoesNotOverwriteADefaultOutputChangedSince(t *testing.T) {
	r := scheduleWriteRig(t)
	for name, v := range map[string][2]any{"EnumSch": {1, "2"}, "NumSch": {5, "6.0"}} {
		out := r.Srv.RunWrite("n4_set_schedule", setSchedule(name, Args{"default_output": v[0]}))
		r.Station.Do(func(st *boxstation.Station) {
			st.Root().Child("Folder").Child(name).Child("defaultOutput").Child("value").Value = v[1]
		})
		if text := r.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]}); !strings.Contains(text, "changed since batch") {
			t.Errorf("%s: %q", name, text)
		}
	}
}

func TestVerifySetScheduleRollbackReportsADefaultOutputTheStationDidNotRestore(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("EnumSch", Args{"default_output": 3}))
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		st.Root().Child("Folder").Child("EnumSch").Child("defaultOutput").Child("value").Value = "2"
	}})
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] == "verified" {
		t.Errorf("rollback = %v", back)
	}
}
