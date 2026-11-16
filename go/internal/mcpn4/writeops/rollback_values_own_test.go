package writeops

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func dayOf(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// rangeJSON is one TimeSchedule; ev is its effectiveValue's children ("" for none).
func rangeJSON(start, ev string) string {
	out := `{"n":"time","t":"schedule:TimeSchedule","s":[{"n":"start","t":"baja:Time","v":"` + start + `"}`
	if ev != "" {
		out += `,{"n":"effectiveValue","t":"baja:StatusEnum","f":"o1","s":[` + ev + `]}`
	}
	return out + `]}`
}

// A day's text compares the ranges a rollback writes and reads back: the defaults a station
// leaves out, an enum's range suffix and the order of the ranges do not count; a value, a status,
// a missing value and a time do.
func TestDayTextComparesWhatADayHolds(t *testing.T) {
	day := func(ranges ...string) string {
		s := `{"t":"schedule:DaySchedule","s":[`
		for i, r := range ranges {
			if i > 0 {
				s += ","
			}
			s += r
		}
		return dayText(dayOf(t, s+`]}`))
	}
	sent := day(rangeJSON("08:00:00.000", `{"n":"value","v":"2"},{"n":"status","t":"baja:Status","v":"0"}`),
		rangeJSON("00:00:00.000", `{"n":"value","v":"0"},{"n":"status","v":"0"}`))
	loaded := day(rangeJSON("00:00:00.000", `{"n":"status","v":"0"}`), rangeJSON("08:00:00.000", `{"n":"value","v":"2@{a=1,b=2}"}`))
	if sent != loaded {
		t.Errorf("sent %q != loaded %q", sent, loaded)
	}
	for name, other := range map[string]string{
		"value":    day(rangeJSON("08:00:00.000", `{"n":"value","v":"3"}`), rangeJSON("00:00:00.000", "{}")),
		"status":   day(rangeJSON("08:00:00.000", `{"n":"value","v":"2"},{"n":"status","v":"40"}`), rangeJSON("00:00:00.000", "{}")),
		"no value": day(rangeJSON("08:00:00.000", ""), rangeJSON("00:00:00.000", "{}")),
		"time":     day(rangeJSON("09:00:00.000", `{"n":"value","v":"2"}`), rangeJSON("00:00:00.000", "{}")),
	} {
		if other == loaded {
			t.Errorf("a changed %s compares equal: %q", name, other)
		}
	}
}

func TestObserveOwnSeesADayOfAnotherTypeAsChanged(t *testing.T) {
	nodes := dayOf(t, `{"day": {"t": "schedule:DailySchedule", "s": []}}`)
	if v, ok := observeOwn(nodes, "day", daySchedule); !ok || v != nil {
		t.Errorf("observeOwn = %v %v", v, ok)
	}
}
