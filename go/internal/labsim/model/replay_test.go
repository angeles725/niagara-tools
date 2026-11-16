package model

import (
	"testing"
	"time"
)

// A recorded series: 10 at 0 s, 20 at 60 s, 30 at 120 s.
var (
	recordedAt     = []time.Duration{0, 60 * time.Second, 120 * time.Second}
	recordedValues = []float64{10, 20, 30}
)

func TestReplayHoldsEachRecordUntilTheNextAtTheSpeedUp(t *testing.T) {
	t.Parallel()
	// Speed 60: one recorded minute passes in one second of the simulator.
	g := Replay(recordedAt, recordedValues, 60, 0, false)
	for at, want := range map[time.Duration]float64{
		0: 10, 999 * ms: 10, 1000 * ms: 20, 1999 * ms: 20, 2000 * ms: 30,
	} {
		if got := g.At(at); got != want {
			t.Errorf("replay at %v = %v, want %v", at, got, want)
		}
	}
}

func TestReplayHoldsTheLastRecordAfterTheSeriesEnds(t *testing.T) {
	t.Parallel()
	g := Replay(recordedAt, recordedValues, 1, 0, false)
	for _, at := range []time.Duration{120 * time.Second, 180 * time.Second, 24 * time.Hour} {
		if got := g.At(at); got != 30 {
			t.Errorf("hold_last at %v = %v, want the last record 30", at, got)
		}
	}
}

func TestReplayLoopStartsAgainAfterTheLastInterval(t *testing.T) {
	t.Parallel()
	// The loop is the span (120 s) plus the last interval (60 s): the last record
	// holds as long as the one before it, then the first comes back.
	g := Replay(recordedAt, recordedValues, 1, 0, true)
	for at, want := range map[time.Duration]float64{
		120 * time.Second: 30, 179 * time.Second: 30, 180 * time.Second: 10,
		240 * time.Second: 20, 360 * time.Second: 10,
	} {
		if got := g.At(at); got != want {
			t.Errorf("loop at %v = %v, want %v", at, got, want)
		}
	}
}

func TestReplayStartOffsetSkipsIntoTheSeries(t *testing.T) {
	t.Parallel()
	g := Replay(recordedAt, recordedValues, 1, 90*time.Second, false)
	for at, want := range map[time.Duration]float64{0: 20, 29 * time.Second: 20, 30 * time.Second: 30} {
		if got := g.At(at); got != want {
			t.Errorf("offset 90 s, at %v = %v, want %v", at, got, want)
		}
	}
	looped := Replay(recordedAt, recordedValues, 1, 90*time.Second, true)
	if got := looped.At(90 * time.Second); got != 10 {
		t.Errorf("offset 90 s looped, at 90 s = %v, want the first record again", got)
	}
}

func TestReplayOfOneRecordIsConstant(t *testing.T) {
	t.Parallel()
	for _, loop := range []bool{false, true} {
		g := Replay([]time.Duration{0}, []float64{7}, 10, 0, loop)
		if g.At(0) != 7 || g.At(time.Hour) != 7 {
			t.Errorf("one record, loop %v: %v %v, want 7", loop, g.At(0), g.At(time.Hour))
		}
	}
}
