package blackbox

// Helpers of the schedule tests: a fake station holding the six schedules recorded live on a
// 4.15.6.20 lab station (testdata/schedules.json) and that station's schedule type registry.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

type scheduleFixture struct {
	Types     map[string]any            `json:"types"`
	Schedules map[string]map[string]any `json:"schedules"`
}

func loadScheduleFixture(t testing.TB) scheduleFixture {
	t.Helper()
	data, err := os.ReadFile("testdata/schedules.json")
	if err != nil {
		t.Fatal(err)
	}
	var f scheduleFixture
	if err := json.Unmarshal(data, &f); err != nil {
		t.Fatal(err)
	}
	return f
}

// scheduleRig is a connected rig whose Folder holds the recorded schedules (BoolSch, NumSch,
// EnumSch, StrSch, Cal, TrigSch) and whose registry describes the schedule types; registry false
// leaves the types undescribed. configure, when not nil, edits the options (writes).
func scheduleRig(t *testing.T, registry bool, configure func(r *Rig, o *Options)) *Rig {
	t.Helper()
	f := loadScheduleFixture(t)
	r := NewRig(t, configure)
	if configure != nil {
		r.ConnectVerified()
	} else {
		r.Connect(nil)
	}
	if registry {
		r.Station.SetRegistry(f.Types)
	}
	for _, name := range []string{"BoolSch", "NumSch", "EnumSch", "StrSch", "Cal", "TrigSch"} {
		body := f.Schedules[name]
		install(r, r.Add("3", name, body["t"].(string)).Handle, body)
	}
	return r
}

// install adds the slots of a BOX body under the component with handle h: components through
// the fake's add (so each gets a handle, as on a station), everything else through a set.
func install(r *Rig, h string, body map[string]any) {
	subs, _ := body["s"].([]any)
	for _, s := range subs {
		slot := s.(map[string]any)
		name, typ := slot["n"].(string), fmt.Sprint(slot["t"])
		if strings.HasPrefix(typ, "schedule:") {
			install(r, r.Add(h, name, typ).Handle, slot)
		} else {
			r.Set(h, name, slot)
		}
	}
}

// readSchedule is the n4_read_schedule reply for the Folder child name.
func readSchedule(r *Rig, name string) map[string]any {
	r.T.Helper()
	return r.Srv.OK("n4_read_schedule", Args{"ord": Folder + "/" + name})
}

// jsonOf is v as compact JSON, for comparing decoded shapes.
func jsonOf(t testing.TB, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// canon is the JSON text want re-encoded with sorted keys, as jsonOf writes a decoded reply.
func canon(t testing.TB, want string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(want), &v); err != nil {
		t.Fatalf("bad expectation %s: %v", want, err)
	}
	return jsonOf(t, v)
}

// handleAt is the handle of the node at slash path below the Folder child name.
func handleAt(r *Rig, name, path string) (h string) {
	r.T.Helper()
	r.Station.Do(func(st *boxstation.Station) {
		n := st.ByHandle("3").Child(name)
		for _, p := range strings.Split(path, "/") {
			if n == nil {
				break
			}
			n = n.Child(p)
		}
		if n != nil {
			h = n.Handle
		}
	})
	if h == "" {
		r.T.Fatalf("no node %s/%s", name, path)
	}
	return h
}

// timeRange is the BOX body of a TimeSchedule child; value "" leaves effectiveValue out, status
// "" means ok.
func timeRange(name, start, finish, valueType, value, status string) map[string]any {
	s := []any{map[string]any{"nm": "p", "n": "start", "t": "baja:Time", "v": start},
		map[string]any{"nm": "p", "n": "finish", "t": "baja:Time", "v": finish}}
	if value != "" {
		kids := []any{map[string]any{"nm": "p", "n": "value", "v": value}}
		if status != "" {
			kids = append(kids, map[string]any{"nm": "p", "n": "status", "t": "baja:Status", "v": status})
		}
		s = append(s, map[string]any{"nm": "p", "n": "effectiveValue", "t": valueType, "f": "o1", "s": kids})
	}
	return map[string]any{"nm": "p", "n": name, "t": "schedule:TimeSchedule", "s": s}
}

// daySchedule is the BOX body of a DaySchedule holding ranges.
func daySchedule(ranges ...any) map[string]any {
	return map[string]any{"nm": "p", "t": "schedule:DaySchedule", "s": ranges}
}
