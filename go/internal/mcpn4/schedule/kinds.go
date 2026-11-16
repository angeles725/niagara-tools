// Package schedule is the station-free model of Niagara schedules: the kinds that exist, how a
// schedule component tree loaded over BOX decodes into plain weekly ranges, special events and
// dates, and how a plain request encodes back into BOX bodies. It never talks to a station.
//
// Evidence: schedule-rt 4.15 (javax.baja.schedule), the Scheduling guide and live probes on a
// 4.15.6.20 station (odd/tasks note of the schedules unit).
package schedule

// Kind is one schedule component kind a station can hold.
type Kind struct {
	Name, Type string
	// ValueType is the Status type of out, defaultOutput and each range value ("" for trigger).
	ValueType string
	// Value is how a value is given and returned in plain JSON.
	Value                     string
	DefaultOutput             string
	Weekly, SpecialEvents     bool // has a weekly schedule and special events
	Dates, Times              bool // has calendar dates (calendar, trigger) / trigger times
	Purpose, Behaviour, Links string
}

// Kinds are the six schedule kinds, in the order the tools list them.
var Kinds = []Kind{
	{Name: "boolean", Type: "schedule:BooleanSchedule", ValueType: "baja:StatusBoolean", Value: "true or false",
		DefaultOutput: "false", Weekly: true, SpecialEvents: true,
		Purpose:   "on/off control by time of day: occupancy, lighting, a fan enable",
		Behaviour: weeklyBehaviour, Links: "out to a BooleanWritable input, by convention in16"},
	{Name: "numeric", Type: "schedule:NumericSchedule", ValueType: "baja:StatusNumeric", Value: "a number",
		DefaultOutput: "null", Weekly: true, SpecialEvents: true,
		Purpose:   "a number by time of day: a setpoint that changes between occupied and unoccupied",
		Behaviour: weeklyBehaviour, Links: "out to a NumericWritable input, by convention in16"},
	{Name: "enum", Type: "schedule:EnumSchedule", ValueType: "baja:StatusEnum",
		Value: "an ordinal (integer) of the schedule's range facet", DefaultOutput: "null", Weekly: true, SpecialEvents: true,
		Purpose:   "one of several named states by time of day: occupied / unoccupied / standby",
		Behaviour: weeklyBehaviour + " Set the range facet before adding events: values are its ordinals.",
		Links:     "out to an EnumWritable input with the same range, by convention in16"},
	{Name: "string", Type: "schedule:StringSchedule", ValueType: "baja:StatusString", Value: "a string",
		DefaultOutput: "null", Weekly: true, SpecialEvents: true,
		Purpose:   "a text by time of day: a mode or message name",
		Behaviour: weeklyBehaviour, Links: "out to a StringWritable input, by convention in16"},
	{Name: "calendar", Type: "schedule:CalendarSchedule", ValueType: "baja:StatusBoolean", Value: "none (dates only)",
		DefaultOutput: "false", Dates: true,
		Purpose: "a list of days (holidays, shutdowns) that weekly schedules reference from a special event",
		Behaviour: "out is true on any day one of its dates matches, false otherwise; it has no times. " +
			"Editing it changes the days of every reference special event that points to it.",
		Links: "usually not linked: reference it from a weekly schedule's special event"},
	{Name: "trigger", Type: "schedule:TriggerSchedule", Dates: true, Times: true, Value: "none (times only)",
		Purpose: "fires its trigger topic at given times of day on the days its dates match (or every day)",
		Behaviour: "no out value: at each time it fires the trigger topic, which a link carries to an " +
			"action (e.g. a history archive or a reset); dates is an embedded calendar, times the firing times.",
		Links: "trigger topic to an action of another component"},
}

const weeklyBehaviour = "out follows, highest first: a non-null in; the default output when the " +
	"effective period does not include now; the highest active special event (top of the list); the " +
	"active weekly range; the default output. A range runs from start (inclusive) to end (exclusive); " +
	"out, next time and next value are recalculated by the station after every change."

// EventKind is one kind of special event (or calendar date) and its rule fields.
type EventKind struct {
	Name, Type, Purpose string
	Fields              []Field
}

// Field is one rule field of an event kind.
type Field struct{ Name, Values string }

// EventKinds are the five special event kinds; a calendar holds the first four as its dates.
var EventKinds = []EventKind{
	{Name: "date", Type: "schedule:DateSchedule", Purpose: "one day, or every day matching the fields given " +
		"(a field left out or \"*\" matches any)", Fields: []Field{yearField, monthField, dayField, weekdayField}},
	{Name: "date_range", Type: "schedule:DateRangeSchedule", Purpose: "every day from start to end inclusive",
		Fields: []Field{{"start", "a date rule {year, month, day} (fields may be \"*\")"}, {"end", "a date rule like start"}}},
	{Name: "week_and_day", Type: "schedule:WeekAndDaySchedule", Purpose: "a weekday in a week of a month, " +
		"e.g. the 4th Thursday of November", Fields: []Field{monthField, weekField, weekdayField}},
	{Name: "custom", Type: "schedule:CustomSchedule", Purpose: "any combination of years, months, days, weeks " +
		"and weekdays; each list matches any of its entries, an empty or absent list matches all",
		Fields: []Field{yearField, {"months", "list of month names"}, {"days", "list of 1-31"},
			{"weeks", "list of week names"}, {"weekdays", "list of weekday names"}}},
	{Name: "reference", Type: "schedule:ScheduleReference", Purpose: "the days of a CalendarSchedule; " +
		"editing that calendar changes this event", Fields: []Field{{"calendar", "the calendar's slot ORD"}}},
}

var (
	yearField    = Field{"year", "a year such as 2026, or \"*\" for every year"}
	monthField   = Field{"month", "jan..dec, odd (jan, mar, may, jul, sep, nov), even, or \"*\""}
	dayField     = Field{"day", "1..31 or \"*\""}
	weekdayField = Field{"weekday", "sunday..saturday or \"*\""}
	weekField    = Field{"week", "week1..week5 (days 1-7, 8-14, ...), last7days, calendarWeek1..calendarWeek6, " +
		"before_last7, before_last14, before_last21, or \"*\""}
)

// Days are the weekly schedule's day slot names, in the order the tools return them.
var Days = []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
