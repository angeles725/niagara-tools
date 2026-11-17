// Package refrig holds the refrigeration fault rules of n4_refrigeration_faults (opportunity
// O1): pure functions over history series, every threshold a parameter. Nothing here reads a
// station; the read tool fetches the series and formats the findings.
package refrig

import (
	"sort"
	"time"
)

// Sample is one history record: its time and value (a boolean history is 1 or 0).
type Sample struct {
	T time.Time
	V float64
}

// Series is a history in time order (oldest first).
type Series []Sample

// Sorted returns s in time order.
func Sorted(s Series) Series {
	out := append(Series(nil), s...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].T.Before(out[j].T) })
	return out
}

// Interval is one period a boolean series held true: [Start, End). Open is true when the series
// ended while still true (End is then the window end).
type Interval struct {
	Start, End time.Time
	Open       bool
}

// Duration is End - Start.
func (i Interval) Duration() time.Duration { return i.End.Sub(i.Start) }

// OnIntervals reads a boolean series as a step function: true from a sample whose value is
// non-zero until the next sample that is zero; a period still true at the last sample ends at
// windowEnd and is Open. A period already true at the first sample starts there (its real start
// is before the window).
func OnIntervals(s Series, windowEnd time.Time) []Interval {
	var out []Interval
	on := false
	var start time.Time
	for _, x := range s {
		switch {
		case x.V != 0 && !on:
			on, start = true, x.T
		case x.V == 0 && on:
			on = false
			out = append(out, Interval{Start: start, End: x.T})
		}
	}
	if on {
		out = append(out, Interval{Start: start, End: windowEnd, Open: true})
	}
	return out
}

// At is the value of s in force at t (the last sample at or before t); false when none is.
func At(s Series, t time.Time) (float64, bool) {
	i := sort.Search(len(s), func(i int) bool { return s[i].T.After(t) })
	if i == 0 {
		return 0, false
	}
	return s[i-1].V, true
}

// Max is the largest value of s within [from, to], with its time; false when no sample falls in.
func Max(s Series, from, to time.Time) (Sample, bool) {
	var best Sample
	found := false
	for _, x := range s {
		if x.T.Before(from) || x.T.After(to) {
			continue
		}
		if !found || x.V > best.V {
			best, found = x, true
		}
	}
	return best, found
}
