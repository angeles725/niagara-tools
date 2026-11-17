package refrig

import (
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)

// at is t0 plus m minutes.
func at(m float64) time.Time { return t0.Add(time.Duration(m * float64(time.Minute))) }

// steps builds a series from (minute, value) pairs.
func steps(pairs ...float64) Series {
	var s Series
	for i := 0; i < len(pairs); i += 2 {
		s = append(s, Sample{at(pairs[i]), pairs[i+1]})
	}
	return s
}

func TestOnIntervalsReadsAStepFunctionAndAnOpenEnd(t *testing.T) {
	got := OnIntervals(steps(0, 1, 5, 1, 10, 0, 20, 1), at(30))
	if len(got) != 2 || got[0].Start != at(0) || got[0].End != at(10) || got[0].Open ||
		got[1].Start != at(20) || got[1].End != at(30) || !got[1].Open {
		t.Fatalf("%+v", got)
	}
}

func TestShortCyclingFlagsShortRunsAndBusyHours(t *testing.T) {
	// runs: 0-1 (1 min), 10-25 (15 min), 30-31.5 (1.5 min), 50-open
	run := steps(0, 1, 1, 0, 10, 1, 25, 0, 30, 1, 31.5, 0, 50, 1)
	short, busy := ShortCycling(run, at(60), 2*time.Minute, 3, time.UTC)
	if len(short) != 2 || short[0].Run.Start != at(0) || short[1].Run.Start != at(30) {
		t.Fatalf("short %+v", short)
	}
	if len(busy) != 1 || busy[0].Starts != 4 || busy[0].Hour != t0 {
		t.Fatalf("busy %+v", busy)
	}
	if _, busy := ShortCycling(run, at(60), time.Minute, 4, time.UTC); len(busy) != 0 {
		t.Errorf("4 starts allowed: %+v", busy)
	}
}

func TestDefrostTerminationTellsTemperatureTimeEarlyAndOpen(t *testing.T) {
	defrost := steps(0, 1, 20, 0, 60, 1, 105, 0, 200, 1, 205, 0, 300, 1)
	coil := steps(0, -20, 15, 12, 60, -20, 100, 4, 200, -20, 205, -18, 300, -20)
	got := DefrostTermination(defrost, coil, at(320), 10, 45*time.Minute, 2*time.Minute)
	want := []string{"temperature", "time", "early", "open"}
	if len(got) != 4 {
		t.Fatalf("%+v", got)
	}
	for i, w := range want {
		if got[i].EndedBy != w {
			t.Errorf("cycle %d ended by %s, want %s (%+v)", i, got[i].EndedBy, w, got[i])
		}
	}
	if !got[1].HasTemp || got[1].EndTemp != 4 || got[1].MaxTemp != 4 {
		t.Errorf("time cycle temps %+v", got[1])
	}
}

func TestSetpointNotReachedNeedsTheMinimumTimeAndSkipsExcludedPeriods(t *testing.T) {
	temp := steps(0, 2, 10, 6, 20, 7, 40, 3, 50, 8, 55, 2, 70, 9, 80, 9)
	defrost := []Interval{{Start: at(65), End: at(75)}}
	got := SetpointNotReached(temp, at(90), 2, 2, 20*time.Minute, defrost)
	// 10-40: 30 min above 4 (peak 7 at 20) counts; 50-55: 5 min does not; 70 is excluded; 80-90 open: 10 min does not.
	if len(got) != 1 || got[0].Period.Start != at(10) || got[0].Period.End != at(40) || got[0].Peak.V != 7 {
		t.Fatalf("%+v", got)
	}
	long := SetpointNotReached(steps(0, 9), at(60), 2, 2, 30*time.Minute, nil)
	if len(long) != 1 || !long[0].Period.Open {
		t.Errorf("open excursion %+v", long)
	}
}

func TestOverAmpsGroupsByHourWithPeakAndCount(t *testing.T) {
	amps := steps(0, 10, 30, 21, 45, 25, 70, 12, 130, 22)
	got := OverAmps(amps, 20, time.UTC)
	if len(got) != 2 || got[0].Hour != t0 || got[0].Over != 2 || got[0].Count != 3 || got[0].Peak.V != 25 ||
		got[1].Hour != at(120) || got[1].Over != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestDoorDriftMeasuresTheRiseUntilCloseAndAfter(t *testing.T) {
	door := steps(0, 1, 5, 0, 30, 1, 32, 0)
	temp := steps(0, 2, 4, 4, 8, 6.5, 30, 2, 33, 3)
	got := DoorDrift(door, temp, at(60), 3, 5*time.Minute)
	if len(got) != 1 || got[0].StartTemp != 2 || got[0].Peak.V != 6.5 || got[0].Rise != 4.5 {
		t.Fatalf("%+v", got)
	}
}
