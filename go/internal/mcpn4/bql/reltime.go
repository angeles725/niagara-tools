package bql

import (
	"regexp"
	"strconv"
	"time"
)

// Time bounds of the alarm and history reads (retro 2026-11-16 M2): a BAbsTime text, or a span
// back from now such as 24h, so "since yesterday" needs no date arithmetic by the caller.

// Now is the wall clock a relative bound counts back from (tests replace it).
var Now = time.Now

// relTimeRe is a relative bound: 1 to 5 digits and a unit (m minutes, h hours, d days).
var relTimeRe = regexp.MustCompile(`^([0-9]{1,5})([mhd])$`)

// relUnits are the units of a relative bound.
var relUnits = map[string]time.Duration{"m": time.Minute, "h": time.Hour, "d": 24 * time.Hour}

// absTimeLayout reads a BAbsTime text; utcLayout writes a resolved relative bound.
const (
	absTimeLayout = "2006-01-02T15:04:05.999999999Z07:00"
	utcLayout     = "2006-01-02T15:04:05.000Z"
)

// ResolveTime is value as BAbsTime text and epoch milliseconds: a BAbsTime as given, or a
// relative span (24h, 90m, 7d; never 0) as now minus the span in UTC. ok is false for any
// other text: the caller names the argument in its own message.
func ResolveTime(value string) (string, int64, bool) {
	if m := relTimeRe.FindStringSubmatch(value); m != nil {
		n, _ := strconv.Atoi(m[1])
		if n == 0 {
			return "", 0, false
		}
		at := Now().Add(-time.Duration(n) * relUnits[m[2]]).UTC()
		return at.Format(utcLayout), at.UnixMilli(), true
	}
	if !absTimeRe.MatchString(value) {
		return "", 0, false
	}
	t, err := time.Parse(absTimeLayout, value)
	if err != nil {
		return "", 0, false
	}
	return value, t.UnixMilli(), true
}

// IsoOrNull is a BAbsTime's encodeToString text, or nil for the null time (epoch 0, which a
// station renders in its own zone, e.g. 1969-12-31T18:00:00.000-06:00) or an empty cell.
func IsoOrNull(text string) any {
	if text == "" || text == "null" {
		return nil
	}
	if t, err := time.Parse(absTimeLayout, text); err == nil && t.UnixMilli() == 0 {
		return nil
	}
	return text
}
