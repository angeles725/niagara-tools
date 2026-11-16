package schedule

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func compact(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Canonical)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A day is sent as the station stores it (live 4.15.6.20): ranges named time, time1, ...,
// midnight as 00:00, the value in an effectiveValue with flags o1.
func TestDayBodyIsADayScheduleOfNamedTimeRanges(t *testing.T) {
	k := KindOf("schedule:NumericSchedule")
	day := DayBody(k, []Range{{480, 720, pyjson.FloatNumber(21.5)}, {780, 1440, pyjson.IntLiteral("19")}})
	want := `{"nm":"p","s":[` +
		`{"n":"time","nm":"p","s":[{"n":"start","nm":"p","t":"baja:Time","v":"08:00:00.000"},` +
		`{"n":"finish","nm":"p","t":"baja:Time","v":"12:00:00.000"},{"f":"o1","n":"effectiveValue","nm":"p","s":[` +
		`{"n":"value","nm":"p","t":"baja:Double","v":"21.5"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],` +
		`"t":"baja:StatusNumeric"}],"t":"schedule:TimeSchedule"},` +
		`{"n":"time1","nm":"p","s":[{"n":"start","nm":"p","t":"baja:Time","v":"13:00:00.000"},` +
		`{"n":"finish","nm":"p","t":"baja:Time","v":"00:00:00.000"},{"f":"o1","n":"effectiveValue","nm":"p","s":[` +
		`{"n":"value","nm":"p","t":"baja:Double","v":"19.0"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],` +
		`"t":"baja:StatusNumeric"}],"t":"schedule:TimeSchedule"}],"t":"schedule:DaySchedule"}`
	if got := compact(t, day); got != want {
		t.Errorf("day =\n%s\nwant\n%s", got, want)
	}
	if got := compact(t, DayBody(k, nil)); got != `{"nm":"p","s":[],"t":"schedule:DaySchedule"}` {
		t.Errorf("empty day = %s", got)
	}
}

func TestStatusBodyEncodesEachKindAndNull(t *testing.T) {
	for typ, c := range map[string]struct {
		v    any
		want string
	}{
		"BooleanSchedule": {true, `"t":"baja:Boolean","v":"true"`},
		"EnumSchedule":    {pyjson.IntLiteral("2"), `"v":"2"`},
		"StringSchedule":  {"eco", `"t":"baja:String","v":"eco"`},
		"NumericSchedule": {nil, `"t":"baja:Double","v":"0.0"`},
	} {
		k := KindOf("schedule:" + typ)
		status := "0"
		if c.v == nil {
			status = "40"
		}
		want := `{"nm":"p","s":[{"n":"value","nm":"p",` + c.want + `},{"n":"status","nm":"p","t":"baja:Status","v":"` +
			status + `"}],"t":"` + k.ValueType + `"}`
		if got := compact(t, StatusBody(k, c.v)); got != want {
			t.Errorf("%s =\n%s\nwant\n%s", typ, got, want)
		}
	}
}

// Strip keeps what a set needs to put a loaded subtree back, and drops the handles, display
// strings and the top node's own name.
func TestStripKeepsOnlyTheBodyOfALoadedSubtree(t *testing.T) {
	loaded, _ := pyjson.Unmarshal([]byte(`{"nm":"p","n":"day","h":"9","d":"Day","t":"schedule:DaySchedule","s":[
		{"nm":"p","n":"time","h":"10","t":"schedule:TimeSchedule","s":[{"n":"start","t":"baja:Time","v":"08:00:00.000","d":"8"},
		{"n":"effectiveValue","f":"o1","t":"baja:StatusBoolean","s":[]}]}]}`))
	want := `{"nm":"p","s":[{"n":"time","nm":"p","s":[{"n":"start","t":"baja:Time","v":"08:00:00.000"},` +
		`{"f":"o1","n":"effectiveValue","s":[],"t":"baja:StatusBoolean"}],"t":"schedule:TimeSchedule"}],` +
		`"t":"schedule:DaySchedule"}`
	if got := compact(t, Strip(loaded.(*pyjson.Object))); got != want {
		t.Errorf("strip =\n%s\nwant\n%s", got, want)
	}
}

func TestViewAndStatusViewMatchTheReadModel(t *testing.T) {
	k := KindOf("schedule:EnumSchedule")
	if got := compact(t, View(k, Range{390, 1440, pyjson.IntLiteral("3")})); got != `{"end":"24:00","start":"06:30","value":3}` {
		t.Errorf("view = %s", got)
	}
	if got := compact(t, StatusView(k, nil)); got != `{"status":"null","value":null}` {
		t.Errorf("null = %s", got)
	}
}
