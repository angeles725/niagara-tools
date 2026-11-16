package schedule

// The ordinals of the station's day-rule sets (schedule-rt ScheduleUtil ranges; the week-of-month
// ordinals were read from a 4.15.6.20 station's type registry, WeekOfMonthSchedule.set "x").

// Months are MonthSchedule.set ordinals 0-13: the twelve months, then the odd and even months.
var Months = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
	"odd", "even"}

// Weekdays are WeekdaySchedule.set ordinals 0-6 (BWeekday: sunday is 0).
var Weekdays = []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// Weeks are WeekOfMonthSchedule.set ordinals 1-15; ordinal 0 is "any week" ("*").
var Weeks = []string{"*", "week1", "week2", "week3", "week4", "week5", "last7days",
	"calendarWeek1", "calendarWeek2", "calendarWeek3", "calendarWeek4", "calendarWeek5", "calendarWeek6",
	"before_last7", "before_last14", "before_last21"}

// MaxDay is the highest DayOfMonthSchedule.set ordinal; ordinal 0 is "any day" ("*").
const MaxDay = 31

// enumName is the name of ordinal n in names, or n itself when the table has no such entry.
func enumName(names []string, n int) any {
	if n >= 0 && n < len(names) {
		return names[n]
	}
	return n
}

// enumOrdinal is the ordinal of name in names, -1 when absent.
func enumOrdinal(names []string, name string) int {
	for i, n := range names {
		if n == name {
			return i
		}
	}
	return -1
}
