package writeops

import (
	"strings"
	"testing"
)

// A rule's text compares what a date or an event's days hold: the defaults a station leaves out,
// the selection mode, an enum set's range suffix and the year of an always-effective
// YearSchedule do not count; a month, a day and a year do.
func TestRuleTextComparesWhatARuleHolds(t *testing.T) {
	rule := func(year, month string, extra string) string {
		return ruleText(dayOf(t, `{"t":"schedule:DateSchedule","s":[
			{"n":"yearSchedule","t":"schedule:YearSchedule","s":[`+year+`]},
			{"n":"monthSchedule","t":"schedule:MonthSchedule","s":[{"n":"set","v":"`+month+`"}`+extra+`]},
			{"n":"daySchedule","t":"schedule:DayOfMonthSchedule"}]}`))
	}
	sent := rule(`{"n":"alwaysEffective","v":"true"},{"n":"year","v":"2026"}`, "0", `,{"n":"singleSelection","v":"true"}`)
	loaded := rule(`{"n":"alwaysEffective","v":"true"}`, "0@{jan=0}", "")
	if sent != loaded {
		t.Errorf("sent %q != loaded %q", sent, loaded)
	}
	fixed := rule(`{"n":"alwaysEffective","v":"false"},{"n":"year","v":"2027"}`, "0", `,{"n":"set","v":""}`)
	if stored := rule(`{"n":"year","v":"2027"}`, "0", ""); fixed != stored {
		t.Errorf("the defaults sent count: %q != %q", fixed, stored)
	}
	for name, other := range map[string]string{
		"month": rule(`{"n":"alwaysEffective","v":"true"}`, "5", ""),
		"year":  rule(`{"n":"year","v":"2027"}`, "0", ""),
	} {
		if other == loaded {
			t.Errorf("a changed %s compares equal: %q", name, other)
		}
	}
}

// The events list compares each event's name, order, ranges and rule.
func TestEventsTextComparesNamesOrderRangesAndRule(t *testing.T) {
	event := func(name, month, start string) string {
		return `{"n":"` + name + `","s":[{"n":"day","t":"schedule:DaySchedule","s":[` + rangeJSON(start, `{"n":"value","v":"true"}`) + `]},
			{"n":"days","t":"schedule:DateSchedule","s":[{"n":"monthSchedule","t":"schedule:MonthSchedule","s":[{"n":"set","v":"` + month + `"}]}]}]}`
	}
	list := func(events ...string) string {
		return eventsText(dayOf(t, `{"t":"schedule:CompositeSchedule","s":[`+strings.Join(events, ",")+`]}`))
	}
	base := list(event("A", "0", "08:00:00.000"), event("B", "1", "08:00:00.000"))
	for name, other := range map[string]string{
		"order":   list(event("B", "1", "08:00:00.000"), event("A", "0", "08:00:00.000")),
		"missing": list(event("A", "0", "08:00:00.000")),
		"rule":    list(event("A", "5", "08:00:00.000"), event("B", "1", "08:00:00.000")),
		"ranges":  list(event("A", "0", "09:00:00.000"), event("B", "1", "08:00:00.000")),
		"name":    list(event("C", "0", "08:00:00.000"), event("B", "1", "08:00:00.000")),
	} {
		if other == base {
			t.Errorf("a changed %s compares equal: %q", name, other)
		}
	}
}

// opDepth loads deep enough to see an own-compared body and leaves every other set as before.
func TestOpDepthReachesOwnComparedContentOnly(t *testing.T) {
	for typ, want := range map[string]int{"schedule:CompositeSchedule": 7, "schedule:MonthSchedule": 3,
		"schedule:DaySchedule": 4, "baja:StatusNumeric": 0} {
		op := dayOf(t, `{"nm":"s","n":"x","b":{"t":"`+typ+`"}}`)
		if got := opDepth(op, "x") - slotDepth("x"); got != want {
			t.Errorf("%s: %d levels below the slot, want %d", typ, got, want)
		}
	}
}
