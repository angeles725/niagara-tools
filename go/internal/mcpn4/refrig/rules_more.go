package refrig

import "time"

// Excursion is one period the room stayed above setpoint + band for at least the minimum time,
// with the highest temperature reached.
type Excursion struct {
	Period Interval
	Peak   Sample
}

// SetpointNotReached finds the periods temp stayed above setpoint+band for at least minDuration
// (step function: a sample's value holds until the next sample). Periods inside any interval of
// exclude (a defrost series, for example) are not counted: a sample during an excluded period
// ends the current excursion and starts none.
func SetpointNotReached(temp Series, windowEnd time.Time, setpoint, band float64, minDuration time.Duration, exclude []Interval) []Excursion {
	limit := setpoint + band
	var out []Excursion
	var cur *Excursion
	closeAt := func(t time.Time, open bool) {
		if cur != nil {
			cur.Period.End, cur.Period.Open = t, open
			if cur.Period.Duration() >= minDuration {
				out = append(out, *cur)
			}
			cur = nil
		}
	}
	for _, x := range temp {
		if x.V <= limit || inAny(x.T, exclude) {
			closeAt(x.T, false)
			continue
		}
		if cur == nil {
			cur = &Excursion{Period: Interval{Start: x.T}, Peak: x}
		} else if x.V > cur.Peak.V {
			cur.Peak = x
		}
	}
	closeAt(windowEnd, true)
	return out
}

func inAny(t time.Time, l []Interval) bool {
	for _, i := range l {
		if !t.Before(i.Start) && t.Before(i.End) {
			return true
		}
	}
	return false
}

// AmpsHour is one clock hour with samples above the limit: their count and the peak.
type AmpsHour struct {
	Hour  time.Time
	Over  int
	Peak  Sample
	Count int
}

// OverAmps groups amps by clock hour (in loc) and returns the hours with at least one sample
// above limit, oldest first.
func OverAmps(amps Series, limit float64, loc *time.Location) []AmpsHour {
	byHour := map[time.Time]*AmpsHour{}
	var order []time.Time
	for _, x := range amps {
		h := x.T.In(loc).Truncate(time.Hour)
		a := byHour[h]
		if a == nil {
			a = &AmpsHour{Hour: h, Peak: x}
			byHour[h] = a
			order = append(order, h)
		}
		a.Count++
		if x.V > a.Peak.V {
			a.Peak = x
		}
		if x.V > limit {
			a.Over++
		}
	}
	var out []AmpsHour
	for _, h := range order {
		if byHour[h].Over > 0 {
			out = append(out, *byHour[h])
		}
	}
	return out
}

// DoorEvent is one door opening whose room temperature rose more than allowed: the temperature
// in force when it opened, the peak until it closed plus the recovery time after, and the rise.
type DoorEvent struct {
	Open      Interval
	StartTemp float64
	Peak      Sample
	Rise      float64
}

// DoorDrift finds the door openings after which the room temperature rose more than maxRise
// above its value at the opening, looking until the door closed plus after (the room keeps
// warming a little after the door shuts).
func DoorDrift(door, temp Series, windowEnd time.Time, maxRise float64, after time.Duration) []DoorEvent {
	var out []DoorEvent
	for _, o := range OnIntervals(door, windowEnd) {
		start, ok := At(temp, o.Start)
		peak, found := Max(temp, o.Start, o.End.Add(after))
		if !ok || !found {
			continue
		}
		if rise := peak.V - start; rise > maxRise {
			out = append(out, DoorEvent{Open: o, StartTemp: start, Peak: peak, Rise: rise})
		}
	}
	return out
}
