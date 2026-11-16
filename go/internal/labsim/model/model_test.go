package model

import (
	"math"
	"testing"
	"time"
)

const ms = time.Millisecond

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestRampClimbsFromMinToMaxAndStartsAgain(t *testing.T) {
	t.Parallel()
	g := Ramp(10, 20, 1000*ms)
	for at, want := range map[time.Duration]float64{0: 10, 250 * ms: 12.5, 500 * ms: 15, 999 * ms: 19.99, 1000 * ms: 10, 2500 * ms: 15} {
		if got := g.At(at); !near(got, want) {
			t.Errorf("ramp at %v = %v, want %v", at, got, want)
		}
	}
}

func TestSineSwingsAroundItsCenter(t *testing.T) {
	t.Parallel()
	g := Sine(50, 5, 4000*ms)
	for at, want := range map[time.Duration]float64{0: 50, 1000 * ms: 55, 2000 * ms: 50, 3000 * ms: 45, 5000 * ms: 55} {
		if got := g.At(at); !near(got, want) {
			t.Errorf("sine at %v = %v, want %v", at, got, want)
		}
	}
}

func TestRandomWalkIsTheSameForTheSameSeedHoweverItIsRead(t *testing.T) {
	t.Parallel()
	walk := func(seed uint64) Generator { return RandomWalk(seed, 50, 2, 100*ms, 40, 60) }
	stepped, jumped, other := walk(7), walk(7), walk(8)
	if got := stepped.At(99 * ms); got != 50 {
		t.Errorf("before the first interval = %v, want the start value 50", got)
	}
	var last float64
	moved := false
	for i := 1; i <= 200; i++ {
		last = stepped.At(time.Duration(i) * 100 * ms)
		if last < 40 || last > 60 {
			t.Fatalf("step %d = %v, outside 40-60", i, last)
		}
		moved = moved || last != 50
	}
	if !moved {
		t.Error("the walk never left its start value")
	}
	if got := jumped.At(20000 * ms); got != last {
		t.Errorf("read once at step 200 = %v, read at every step = %v", got, last)
	}
	if got := jumped.At(20099 * ms); got != last {
		t.Errorf("inside the same interval = %v, want %v", got, last)
	}
	if got := other.At(20000 * ms); got == last {
		t.Errorf("another seed gave the same value %v", got)
	}
}

func device() *Device {
	two := 2
	return NewDevice(DeviceSpec{
		Name: "unit-a",
		Points: []PointSpec{
			{Name: "setpoint", Initial: 21.5, Writable: true},
			{Name: "ramp", Generator: Ramp(0, 100, 1000*ms)},
			{Name: "fragile", Initial: 1, Faults: Faults{FailAfterReads: &two, Delay: 300 * ms}},
		},
		Faults:  Faults{Delay: 100 * ms},
		Offline: []Window{{From: 5000 * ms, To: 6000 * ms}},
	})
}

func TestDeviceGivesEachPointItsValueAtTheElapsedTime(t *testing.T) {
	t.Parallel()
	d := device()
	if v, ok := d.Get("setpoint", 0); !ok || v != 21.5 {
		t.Errorf("setpoint = %v, %v", v, ok)
	}
	if v, _ := d.Get("ramp", 500*ms); !near(v, 50) {
		t.Errorf("ramp at 500ms = %v, want 50", v)
	}
	if _, ok := d.Get("absent", 0); ok {
		t.Error("a point the device does not have was found")
	}
	if got := d.Points(); len(got) != 3 || got[0] != "setpoint" || got[2] != "fragile" {
		t.Errorf("Points() = %v, want the scenario order", got)
	}
	if !d.Writable("setpoint") || d.Writable("ramp") || d.Writable("absent") {
		t.Error("Writable does not follow the specification of each point")
	}
}

func TestATransactionSeesItsOwnWrites(t *testing.T) {
	t.Parallel()
	d := device()
	d.Do(0, func(tx Tx) {
		tx.Set("setpoint", 19)
		if got := tx.Get("setpoint"); got != 19 {
			t.Errorf("inside the transaction = %v, want 19", got)
		}
	})
	if v, _ := d.Get("setpoint", 0); v != 19 {
		t.Errorf("after the transaction = %v, want 19", v)
	}
}

func TestPinHoldsAGeneratedPointAtAValue(t *testing.T) {
	t.Parallel()
	d := device()
	if !d.Pin("ramp", 42) || d.Pin("absent", 1) {
		t.Fatal("Pin must report whether the point exists")
	}
	if v, _ := d.Get("ramp", 700*ms); v != 42 {
		t.Errorf("pinned ramp = %v, want 42 whatever the time", v)
	}
}

func TestOfflineWindowIncludesItsStartAndExcludesItsEnd(t *testing.T) {
	t.Parallel()
	d := device()
	for at, want := range map[time.Duration]bool{4999 * ms: false, 5000 * ms: true, 5999 * ms: true, 6000 * ms: false} {
		if got := d.Offline(at); got != want {
			t.Errorf("Offline(%v) = %v, want %v", at, got, want)
		}
	}
}

func TestDelayIsTheLongestOfTheDeviceAndTheTouchedPoints(t *testing.T) {
	t.Parallel()
	d := device()
	if got := d.Delay([]string{"setpoint"}); got != 100*ms {
		t.Errorf("device delay = %v, want 100ms", got)
	}
	if got := d.Delay([]string{"setpoint", "fragile"}); got != 300*ms {
		t.Errorf("with the slow point = %v, want 300ms", got)
	}
}

func TestAPointFailsAfterItsReadsAndOnlyReadsCount(t *testing.T) {
	t.Parallel()
	d := device()
	for i := range 3 {
		if !d.Admit([]string{"fragile"}, false) {
			t.Fatalf("write %d refused before any read", i)
		}
	}
	for i := range 2 {
		if !d.Admit([]string{"setpoint", "fragile"}, true) {
			t.Fatalf("read %d refused, the point fails after 2", i+1)
		}
	}
	if d.Admit([]string{"fragile"}, true) || d.Admit([]string{"fragile"}, false) {
		t.Error("the point still answers after its 2 reads")
	}
	if !d.Admit([]string{"setpoint"}, true) {
		t.Error("a request that does not touch the failed point was refused")
	}
}

func TestADeviceFailsAfterItsReads(t *testing.T) {
	t.Parallel()
	one := 1
	d := NewDevice(DeviceSpec{Name: "d", Points: []PointSpec{{Name: "p"}}, Faults: Faults{FailAfterReads: &one}})
	if !d.Admit([]string{"p"}, true) {
		t.Fatal("the first read was refused")
	}
	if d.Admit([]string{"p"}, true) || d.Admit(nil, false) {
		t.Error("the device still answers after its 1 read")
	}
	if got := d.Reads(); got != 1 {
		t.Errorf("Reads() = %d, want 1 (a refused read is not counted)", got)
	}
}

func TestAnOverrideHidesTheFieldValueUntilItIsReleased(t *testing.T) {
	t.Parallel()
	d := device()
	commanded := 55.0
	d.Do(0, func(tx Tx) {
		tx.Override("setpoint", &commanded)
		if got, field := tx.Get("setpoint"), tx.Field("setpoint"); got != 55 || field != 21.5 {
			t.Errorf("overridden: value %v, field %v; want 55 over 21.5", got, field)
		}
		tx.Set("setpoint", 19) // a field write lands under the override
		tx.Override("absent", &commanded)
	})
	if v, _ := d.Get("setpoint", 0); v != 55 {
		t.Errorf("a field write showed through the override: %v", v)
	}
	if !d.Pin("setpoint", 18) {
		t.Fatal("Pin refused an overridden point")
	}
	d.Do(0, func(tx Tx) {
		if got, field := tx.Get("setpoint"), tx.Field("setpoint"); got != 55 || field != 18 {
			t.Errorf("pinned under the override: value %v, field %v; want 55 over 18", got, field)
		}
		tx.Override("setpoint", nil)
		if got := tx.Get("setpoint"); got != 18 {
			t.Errorf("released = %v, want the field value 18", got)
		}
	})
	if v, _ := d.Get("ramp", 500*ms); !near(v, 50) {
		t.Errorf("a point that was never overridden = %v, want its behaviour (50)", v)
	}
}
