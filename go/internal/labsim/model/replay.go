package model

import (
	"sort"
	"time"
)

// replay plays a recorded series back; it holds no state between reads.
type replay struct {
	at     []time.Duration // each record's time since the first, ascending
	values []float64
	speed  float64
	offset time.Duration
	period time.Duration // the loop's length; 0 when it does not loop
}

// Replay plays a recorded series: at[i] is the time of record i since the first
// record (ascending, at[0] is 0) and values[i] its value. A record holds until
// the next one, as a history sample does. The series runs speed times faster
// than it was recorded and starts offset into it. At its end it holds the last
// record or, with loop, starts again once the last record has held as long as
// the one before it; one record is a constant.
func Replay(at []time.Duration, values []float64, speed float64, offset time.Duration, loop bool) Generator {
	r := replay{at: at, values: values, speed: speed, offset: offset}
	if n := len(at); loop && n > 1 {
		r.period = at[n-1] + (at[n-1] - at[n-2])
	}
	return r
}

func (r replay) At(elapsed time.Duration) float64 {
	pos := r.offset + time.Duration(float64(elapsed)*r.speed)
	if r.period > 0 {
		pos %= r.period
	}
	// The last record at or before pos; the first record covers anything earlier.
	i := sort.Search(len(r.at), func(i int) bool { return r.at[i] > pos }) - 1
	return r.values[max(i, 0)]
}
