package bacobj

import (
	"math"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

func write(object bacnet.ObjectID, property bacnet.Property, priority uint32, values ...bacnet.Value) bacnet.WriteProperty {
	return bacnet.WriteProperty{Object: object, Ref: bacnet.PropertyRef{Property: property}, Values: values, HasPriority: priority != 0, Priority: priority}
}

func present(t *testing.T, d *Device, object bacnet.ObjectID) bacnet.Value {
	t.Helper()
	v, err := d.Read(object, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, 0)
	if err != nil || len(v) != 1 {
		t.Fatalf("present-value of %v: %+v, %v", object, v, err)
	}
	return v[0]
}

func TestWritesThatAreRefusedSayWhy(t *testing.T) {
	t.Parallel()
	d := bench(false)
	pv := bacnet.PropPresentValue
	cases := []struct {
		name string
		q    bacnet.WriteProperty
		want *bacnet.Error
	}{
		{"an object the device has not", write(bacnet.ObjectID{Type: bacnet.AnalogValue, Instance: 77}, pv, 0, bacnet.Real(1)), bacnet.ErrUnknownObject},
		{"a property the object has not", write(setpoint, bacnet.PropStateText, 0, bacnet.Real(1)), bacnet.ErrUnknownProperty},
		{"an input", write(supply, pv, 0, bacnet.Real(1)), bacnet.ErrWriteAccessDenied},
		{"a value that is not writable", write(locked, pv, 0, bacnet.Enum(0)), bacnet.ErrWriteAccessDenied},
		{"a read-only property", write(setpoint, bacnet.PropObjectName, 0, bacnet.Text("x")), bacnet.ErrWriteAccessDenied},
		{"out-of-service", write(setpoint, bacnet.PropOutOfService, 0, bacnet.Boolean(true)), bacnet.ErrWriteAccessDenied},
		{"a property of the device", write(device, bacnet.PropObjectName, 0, bacnet.Text("x")), bacnet.ErrWriteAccessDenied},
		{"an unsigned into an analog", write(setpoint, pv, 0, bacnet.Unsigned(21)), bacnet.ErrInvalidDataType},
		{"a real into a binary", write(command, pv, 8, bacnet.Real(1)), bacnet.ErrInvalidDataType},
		{"an enumerated into a multi-state", write(season, pv, 0, bacnet.Enum(1)), bacnet.ErrInvalidDataType},
		{"a null into a value that is not commandable", write(setpoint, pv, 0, bacnet.Null()), bacnet.ErrInvalidDataType},
		{"two values", write(setpoint, pv, 0, bacnet.Real(1), bacnet.Real(2)), bacnet.ErrInvalidDataType},
		{"no value", write(setpoint, pv, 0), bacnet.ErrInvalidDataType},
		{"an analog above its maximum", write(damper, pv, 8, bacnet.Real(100.5)), bacnet.ErrValueOutOfRange},
		{"an analog below its minimum", write(damper, pv, 8, bacnet.Real(-1)), bacnet.ErrValueOutOfRange},
		{"a number that is not one", write(setpoint, pv, 0, bacnet.Real(float32(math.NaN()))), bacnet.ErrValueOutOfRange},
		{"a binary state above active", write(command, pv, 8, bacnet.Enum(2)), bacnet.ErrValueOutOfRange},
		{"state 0", write(season, pv, 0, bacnet.Unsigned(0)), bacnet.ErrValueOutOfRange},
		{"a state past the last", write(season, pv, 0, bacnet.Unsigned(4)), bacnet.ErrValueOutOfRange},
		{"priority 0", bacnet.WriteProperty{Object: damper, Ref: bacnet.PropertyRef{Property: pv}, Values: []bacnet.Value{bacnet.Real(1)}, HasPriority: true}, bacnet.ErrValueOutOfRange},
		{"priority 17", write(damper, pv, 17, bacnet.Real(1)), bacnet.ErrValueOutOfRange},
		{"an index into present-value", bacnet.WriteProperty{Object: damper, Ref: bacnet.PropertyRef{Property: pv, Index: at(1)}, Values: []bacnet.Value{bacnet.Real(1)}}, bacnet.ErrNotAnArray},
		{"the priority array itself", write(damper, bacnet.PropPriorityArray, 0, bacnet.Real(1)), bacnet.ErrWriteAccessDenied},
		{"an out of range relinquish default", write(damper, bacnet.PropRelinquishDefault, 0, bacnet.Real(101)), bacnet.ErrValueOutOfRange},
	}
	for _, c := range cases {
		if err := d.Write(c.q, 0); err == nil || *err != *c.want {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	if got := present(t, d, damper); got.Real != 35 {
		t.Errorf("a refused write changed the damper to %v", got.Real)
	}
}

func TestAPlainWriteChangesAWritableValue(t *testing.T) {
	t.Parallel()
	d := bench(false)
	for object, value := range map[bacnet.ObjectID]bacnet.Value{setpoint: bacnet.Real(19.5), season: bacnet.Unsigned(1)} {
		// A priority is ignored by an object that is not commandable.
		if err := d.Write(write(object, bacnet.PropPresentValue, 3, value), 0); err != nil {
			t.Fatalf("%v: %v", object, err)
		}
		if got := present(t, d, object); !got.Equal(value) {
			t.Errorf("%v = %+v, want %+v", object, got, value)
		}
	}
	if v, _ := d.Model().Get("setpoint", 0); v != 19.5 {
		t.Errorf("the model point = %v, want 19.5", v)
	}
}

func TestTheHighestPriorityCommandWinsAndRelinquishFallsBack(t *testing.T) {
	t.Parallel()
	d := bench(false)
	pv := bacnet.PropPresentValue
	slot := func(i uint32) bacnet.Value {
		v, err := d.Read(damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray, Index: at(i)}, 0)
		if err != nil {
			t.Fatalf("priority-array[%d]: %v", i, err)
		}
		return v[0]
	}
	steps := []struct {
		name string
		q    bacnet.WriteProperty
		want float32
	}{
		{"with no priority, 16 is used", write(damper, pv, 0, bacnet.Real(40)), 40},
		{"priority 8 wins over 16", write(damper, pv, 8, bacnet.Real(55)), 55},
		{"priority 3 wins over 8", write(damper, pv, 3, bacnet.Real(70)), 70},
		{"a lower priority changes nothing now", write(damper, pv, 10, bacnet.Real(5)), 70},
		{"relinquishing 3 falls back to 8", write(damper, pv, 3, bacnet.Null()), 55},
		{"relinquishing 8 falls back to 10", write(damper, pv, 8, bacnet.Null()), 5},
		{"relinquishing 10 falls back to 16", write(damper, pv, 10, bacnet.Null()), 40},
		{"relinquishing 16 falls back to the relinquish default", write(damper, pv, 16, bacnet.Null()), 35},
		{"relinquishing an empty slot is accepted", write(damper, pv, 1, bacnet.Null()), 35},
	}
	for _, s := range steps {
		if err := d.Write(s.q, 0); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := present(t, d, damper); got.Real != s.want {
			t.Errorf("%s: present-value %v, want %v", s.name, got.Real, s.want)
		}
		if v, _ := d.Model().Get("damper", 0); float32(v) != s.want {
			t.Errorf("%s: the model point shows %v, want the commanded %v", s.name, v, s.want)
		}
	}
	_ = d.Write(write(damper, pv, 8, bacnet.Real(55)), 0)
	if got := slot(8); !got.Equal(bacnet.Real(55)) || !slot(7).IsNull() || !slot(16).IsNull() {
		t.Errorf("priority-array[8] = %+v, want 55 with its neighbours null", got)
	}
	_ = d.Write(write(command, pv, 8, bacnet.Enum(1)), 0)
	_ = d.Write(write(speed, pv, 8, bacnet.Unsigned(3)), 0)
	if !present(t, d, command).Equal(bacnet.Enum(1)) || !present(t, d, speed).Equal(bacnet.Unsigned(3)) {
		t.Error("a commanded binary or multi-state output does not show its command")
	}
}

func TestTheFieldValueIsTheRelinquishDefaultOfACommandableObject(t *testing.T) {
	t.Parallel()
	d := bench(false)
	pv, rd := bacnet.PropPresentValue, bacnet.PropRelinquishDefault
	relinquish := func() bacnet.Value {
		v, _ := d.Read(damper, bacnet.PropertyRef{Property: rd}, 0)
		return v[0]
	}
	d.Model().Pin("damper", 20) // what the control port's set does
	if present(t, d, damper).Real != 20 || relinquish().Real != 20 {
		t.Error("with nothing commanded, a field change must show in present-value and relinquish-default")
	}
	_ = d.Write(write(damper, pv, 8, bacnet.Real(55)), 0)
	d.Model().Pin("damper", 25)
	if present(t, d, damper).Real != 55 || relinquish().Real != 25 {
		t.Errorf("commanded: present-value %v, relinquish-default %v; want 55 over 25", present(t, d, damper).Real, relinquish().Real)
	}
	if err := d.Write(write(damper, rd, 0, bacnet.Real(30)), 0); err != nil {
		t.Fatalf("writing relinquish-default: %v", err)
	}
	_ = d.Write(write(damper, pv, 8, bacnet.Null()), 0)
	if present(t, d, damper).Real != 30 {
		t.Errorf("relinquished = %v, want the written relinquish default 30", present(t, d, damper).Real)
	}
	d.Model().Pin("supply_temp", 22.25) // an input reports the field value
	if present(t, d, supply).Real != 22.25 {
		t.Errorf("an input after a field change = %v, want 22.25", present(t, d, supply).Real)
	}
}
