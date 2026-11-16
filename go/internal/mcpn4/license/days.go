package license

import (
	"strings"
	"time"
)

// Day is a date as written ("2027-03-31"), ok false for "never" and for what is not a date.
func Day(text string) (time.Time, bool) {
	d, err := time.Parse(dateLayout, strings.TrimSpace(text))
	return d, err == nil
}

// Expiry kinds: what an expiry as written is.
const (
	ExpiryNever = iota
	ExpiryDate
	ExpiryUnread // neither "never" nor a date
)

// ExpiryOf says what an expiry as written is.
func ExpiryOf(expires string) int {
	if strings.TrimSpace(expires) == Never {
		return ExpiryNever
	}
	if _, ok := Day(expires); ok {
		return ExpiryDate
	}
	return ExpiryUnread
}

// StationDay is the calendar day of a station time ("2026-11-15T19:05:43.877-06:00"): the day
// the station's own clock shows, whatever its offset. The text must be a whole valid timestamp.
func StationDay(absTime string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, absTime)
	if err != nil {
		return time.Time{}, false
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), true
}

// DaysLeft is the whole days from today to the expiry (0 on its day, negative once past), ok
// false when the expiry is not a date. It counts calendar days, so no span is too long for it.
func DaysLeft(expires string, today time.Time) (int, bool) {
	d, ok := Day(expires)
	if !ok {
		return 0, false
	}
	return dayNumber(d) - dayNumber(today), true
}

// dayNumber is the count of days of t's calendar date from a fixed day (the proleptic
// Gregorian calendar, by whole 400-year eras).
func dayNumber(t time.Time) int {
	y, month, d := t.Date()
	m := int(month)
	if m <= 2 {
		y--
	}
	era := y / 400
	if y < 0 && y%400 != 0 {
		era--
	}
	yearOfEra := y - era*400
	shifted := m - 3
	if m <= 2 {
		shifted = m + 9
	}
	dayOfYear := (153*shifted+2)/5 + d - 1
	return era*146097 + yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
}
