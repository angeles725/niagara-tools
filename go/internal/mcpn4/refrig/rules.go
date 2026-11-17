package refrig

import "time"

// ShortRun is one compressor run shorter than the minimum.
type ShortRun struct {
	Run Interval
}

// BusyHour is one clock hour with more starts than allowed.
type BusyHour struct {
	Hour   time.Time
	Starts int
}

// ShortCycling finds the runs of a compressor run series shorter than minRun (an open run is
// never short: it has not ended) and the clock hours (in loc) with more than maxStarts starts.
func ShortCycling(run Series, windowEnd time.Time, minRun time.Duration, maxStarts int, loc *time.Location) ([]ShortRun, []BusyHour) {
	var short []ShortRun
	starts := map[time.Time]int{}
	var hours []time.Time
	for _, r := range OnIntervals(run, windowEnd) {
		if !r.Open && r.Duration() < minRun {
			short = append(short, ShortRun{r})
		}
		h := r.Start.In(loc).Truncate(time.Hour)
		if starts[h] == 0 {
			hours = append(hours, h)
		}
		starts[h]++
	}
	var busy []BusyHour
	for _, h := range hours {
		if starts[h] > maxStarts {
			busy = append(busy, BusyHour{h, starts[h]})
		}
	}
	return short, busy
}

// Defrost is one defrost cycle and how it ended: "temperature" (the coil reached the
// termination temperature), "time" (it ran to the maximum time without reaching it), "early"
// (it stopped before both), or "open" (still running at the window end). EndTemp is the coil
// temperature in force at the end (HasTemp false when the temperature history has none).
type Defrost struct {
	Cycle   Interval
	EndedBy string
	EndTemp float64
	HasTemp bool
	MaxTemp float64
	HasMax  bool
}

// DefrostTermination classifies every defrost cycle of defrost against the coil temperature
// series: ended by temperature when the highest coil temperature during the cycle reached
// terminateTemp, by time when it did not and the cycle lasted at least maxDuration - slack.
func DefrostTermination(defrost, coil Series, windowEnd time.Time, terminateTemp float64, maxDuration, slack time.Duration) []Defrost {
	var out []Defrost
	for _, c := range OnIntervals(defrost, windowEnd) {
		d := Defrost{Cycle: c}
		d.EndTemp, d.HasTemp = At(coil, c.End)
		if m, ok := Max(coil, c.Start, c.End); ok {
			d.MaxTemp, d.HasMax = m.V, true
		}
		reached := d.HasMax && d.MaxTemp >= terminateTemp
		switch {
		case c.Open:
			d.EndedBy = "open"
		case reached:
			d.EndedBy = "temperature"
		case c.Duration() >= maxDuration-slack:
			d.EndedBy = "time"
		default:
			d.EndedBy = "early"
		}
		out = append(out, d)
	}
	return out
}
