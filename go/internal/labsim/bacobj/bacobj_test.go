package bacobj

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

func f(v float64) *float64 { return &v }
func at(i uint32) *uint32  { return &i }

var (
	device   = bacnet.ObjectID{Type: bacnet.Device, Instance: 1001}
	supply   = bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 1}
	damper   = bacnet.ObjectID{Type: bacnet.AnalogOutput, Instance: 1}
	setpoint = bacnet.ObjectID{Type: bacnet.AnalogValue, Instance: 2}
	status   = bacnet.ObjectID{Type: bacnet.BinaryInput, Instance: 1}
	command  = bacnet.ObjectID{Type: bacnet.BinaryOutput, Instance: 1}
	locked   = bacnet.ObjectID{Type: bacnet.BinaryValue, Instance: 3}
	mode     = bacnet.ObjectID{Type: bacnet.MultiStateInput, Instance: 1}
	speed    = bacnet.ObjectID{Type: bacnet.MultiStateOutput, Instance: 1}
	season   = bacnet.ObjectID{Type: bacnet.MultiStateValue, Instance: 4}
)

// bench is a device with one object of each type over a model device.
func bench(cov bool) *Device {
	states := []string{"Off", "Heat", "Cool"}
	objects := []ObjectSpec{
		{ID: supply, Point: "supply_temp", Description: "Supply air", Units: 62, Min: f(0), Max: f(50), Resolution: f(0.1), COVIncrement: 0.5},
		{ID: damper, Point: "damper", Units: 98, Commandable: true, Min: f(0), Max: f(100)},
		{ID: setpoint, Point: "setpoint", Units: 62, Writable: true},
		{ID: status, Point: "fan_status", ActiveText: "Running", InactiveText: "Stopped"},
		{ID: command, Point: "fan_cmd", Commandable: true},
		{ID: locked, Point: "locked"},
		{ID: mode, Point: "mode", StateTexts: states},
		{ID: speed, Point: "speed", StateTexts: []string{"Low", "Medium", "High"}, Commandable: true},
		{ID: season, Point: "season", StateTexts: states, Writable: true},
	}
	points := model.NewDevice(model.DeviceSpec{Name: "unit-a", Points: []model.PointSpec{
		{Name: "supply_temp", Initial: 18.5}, {Name: "damper", Initial: 35}, {Name: "setpoint", Initial: 21, Writable: true},
		{Name: "fan_status", Initial: 1}, {Name: "fan_cmd"}, {Name: "locked", Initial: 1},
		{Name: "mode", Initial: 2}, {Name: "speed", Initial: 1}, {Name: "season", Initial: 3, Writable: true},
	}})
	d := NewDevice(DeviceSpec{Instance: 1001, Name: "unit-a", VendorID: 999, VendorName: "Lab", Model: "sim", MaxAPDU: 480, COV: cov,
		Description: "bench"}, objects, points)
	d.SetSupported([]int{bacnet.SupportedReadProperty, bacnet.SupportedWhoIs})
	return d
}

func bits(length int, set ...int) bacnet.Value {
	out := make([]bool, length)
	for _, i := range set {
		out[i] = true
	}
	return bacnet.Bits(out...)
}

func TestEveryPropertyReadsAsItsType(t *testing.T) {
	t.Parallel()
	d := bench(true)
	none := bacnet.Bits(false, false, false, false)
	cases := []struct {
		object bacnet.ObjectID
		ref    bacnet.PropertyRef
		want   []bacnet.Value
	}{
		{supply, bacnet.PropertyRef{Property: bacnet.PropObjectIdentifier}, []bacnet.Value{bacnet.OID(supply)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropObjectName}, []bacnet.Value{bacnet.Text("supply_temp")}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropObjectType}, []bacnet.Value{bacnet.Enum(0)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, []bacnet.Value{bacnet.Real(18.5)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropDescription}, []bacnet.Value{bacnet.Text("Supply air")}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropStatusFlags}, []bacnet.Value{none}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropEventState}, []bacnet.Value{bacnet.Enum(0)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropReliability}, []bacnet.Value{bacnet.Enum(0)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropOutOfService}, []bacnet.Value{bacnet.Boolean(false)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropUnits}, []bacnet.Value{bacnet.Enum(62)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropMinPresValue}, []bacnet.Value{bacnet.Real(0)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropMaxPresValue}, []bacnet.Value{bacnet.Real(50)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropResolution}, []bacnet.Value{bacnet.Real(0.1)}},
		{supply, bacnet.PropertyRef{Property: bacnet.PropCOVIncrement}, []bacnet.Value{bacnet.Real(0.5)}},
		{damper, bacnet.PropertyRef{Property: bacnet.PropRelinquishDefault}, []bacnet.Value{bacnet.Real(35)}},
		{damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray, Index: at(0)}, []bacnet.Value{bacnet.Unsigned(16)}},
		{damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray, Index: at(16)}, []bacnet.Value{bacnet.Null()}},
		{status, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, []bacnet.Value{bacnet.Enum(1)}},
		{status, bacnet.PropertyRef{Property: bacnet.PropPolarity}, []bacnet.Value{bacnet.Enum(0)}},
		{status, bacnet.PropertyRef{Property: bacnet.PropActiveText}, []bacnet.Value{bacnet.Text("Running")}},
		{status, bacnet.PropertyRef{Property: bacnet.PropInactiveText}, []bacnet.Value{bacnet.Text("Stopped")}},
		{command, bacnet.PropertyRef{Property: bacnet.PropRelinquishDefault}, []bacnet.Value{bacnet.Enum(0)}},
		{mode, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, []bacnet.Value{bacnet.Unsigned(2)}},
		{mode, bacnet.PropertyRef{Property: bacnet.PropNumberOfStates}, []bacnet.Value{bacnet.Unsigned(3)}},
		{mode, bacnet.PropertyRef{Property: bacnet.PropStateText}, []bacnet.Value{bacnet.Text("Off"), bacnet.Text("Heat"), bacnet.Text("Cool")}},
		{mode, bacnet.PropertyRef{Property: bacnet.PropStateText, Index: at(0)}, []bacnet.Value{bacnet.Unsigned(3)}},
		{mode, bacnet.PropertyRef{Property: bacnet.PropStateText, Index: at(3)}, []bacnet.Value{bacnet.Text("Cool")}},
		{speed, bacnet.PropertyRef{Property: bacnet.PropRelinquishDefault}, []bacnet.Value{bacnet.Unsigned(1)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectIdentifier}, []bacnet.Value{bacnet.OID(device)}},
		{bacnet.ObjectID{Type: bacnet.Device, Instance: bacnet.WildcardInstance}, bacnet.PropertyRef{Property: bacnet.PropObjectIdentifier}, []bacnet.Value{bacnet.OID(device)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectName}, []bacnet.Value{bacnet.Text("unit-a")}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectType}, []bacnet.Value{bacnet.Enum(8)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropSystemStatus}, []bacnet.Value{bacnet.Enum(0)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropVendorName}, []bacnet.Value{bacnet.Text("Lab")}},
		{device, bacnet.PropertyRef{Property: bacnet.PropVendorIdentifier}, []bacnet.Value{bacnet.Unsigned(999)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropModelName}, []bacnet.Value{bacnet.Text("sim")}},
		{device, bacnet.PropertyRef{Property: bacnet.PropProtocolVersion}, []bacnet.Value{bacnet.Unsigned(1)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropProtocolRevision}, []bacnet.Value{bacnet.Unsigned(bacnet.ProtocolRevision)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropServicesSupported}, []bacnet.Value{bits(40, 12, 34)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectTypesSupported}, []bacnet.Value{bits(25, 0, 1, 2, 3, 4, 5, 8, 13, 14, 19)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropMaxAPDULength}, []bacnet.Value{bacnet.Unsigned(480)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropSegmentationSupported}, []bacnet.Value{bacnet.Enum(3)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropAPDUTimeout}, []bacnet.Value{bacnet.Unsigned(3000)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropNumberOfAPDURetries}, []bacnet.Value{bacnet.Unsigned(0)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropDeviceAddressBinding}, nil},
		{device, bacnet.PropertyRef{Property: bacnet.PropDatabaseRevision}, []bacnet.Value{bacnet.Unsigned(1)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropDescription}, []bacnet.Value{bacnet.Text("bench")}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(0)}, []bacnet.Value{bacnet.Unsigned(10)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(1)}, []bacnet.Value{bacnet.OID(device)}},
		{device, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(10)}, []bacnet.Value{bacnet.OID(season)}},
	}
	for _, c := range cases {
		got, err := d.Read(c.object, c.ref, 0)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v %v: %+v, %v; want %+v", c.object, c.ref.Property, got, err, c.want)
		}
	}
	if list, err := d.Read(device, bacnet.PropertyRef{Property: bacnet.PropObjectList}, 0); err != nil || len(list) != 10 || list[0].Object != device || list[2].Object != damper {
		t.Errorf("object-list = %+v, %v; want the device then the objects in scenario order", list, err)
	}
	if list, _ := d.Read(damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray}, 0); len(list) != 16 || !list[7].IsNull() {
		t.Errorf("priority-array = %+v, want 16 nulls", list)
	}
}

func TestReadingWhatIsNotThereGivesItsError(t *testing.T) {
	t.Parallel()
	d := bench(false)
	cases := []struct {
		name   string
		object bacnet.ObjectID
		ref    bacnet.PropertyRef
		want   *bacnet.Error
	}{
		{"an object the device has not", bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 9}, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, bacnet.ErrUnknownObject},
		{"another device", bacnet.ObjectID{Type: bacnet.Device, Instance: 7}, bacnet.PropertyRef{Property: bacnet.PropObjectName}, bacnet.ErrUnknownObject},
		{"a property no object has", supply, bacnet.PropertyRef{Property: 9999}, bacnet.ErrUnknownProperty},
		{"a property of another kind", supply, bacnet.PropertyRef{Property: bacnet.PropStateText}, bacnet.ErrUnknownProperty},
		{"the priority array of an input", supply, bacnet.PropertyRef{Property: bacnet.PropPriorityArray}, bacnet.ErrUnknownProperty},
		{"the relinquish default of a value that is not commandable", setpoint, bacnet.PropertyRef{Property: bacnet.PropRelinquishDefault}, bacnet.ErrUnknownProperty},
		{"cov-increment when the device has no COV", supply, bacnet.PropertyRef{Property: bacnet.PropCOVIncrement}, bacnet.ErrUnknownProperty},
		{"a description that was not given", damper, bacnet.PropertyRef{Property: bacnet.PropDescription}, bacnet.ErrUnknownProperty},
		{"a limit that was not given", setpoint, bacnet.PropertyRef{Property: bacnet.PropMaxPresValue}, bacnet.ErrUnknownProperty},
		{"texts that were not given", locked, bacnet.PropertyRef{Property: bacnet.PropActiveText}, bacnet.ErrUnknownProperty},
		{"the polarity of a binary value", locked, bacnet.PropertyRef{Property: bacnet.PropPolarity}, bacnet.ErrUnknownProperty},
		{"a set of properties", supply, bacnet.PropertyRef{Property: bacnet.PropAll}, bacnet.ErrUnknownProperty},
		{"an index into a scalar", supply, bacnet.PropertyRef{Property: bacnet.PropPresentValue, Index: at(1)}, bacnet.ErrNotAnArray},
		{"an index past the object list", device, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(11)}, bacnet.ErrInvalidArrayIndex},
		{"an index past the priority array", damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray, Index: at(17)}, bacnet.ErrInvalidArrayIndex},
		{"an index past the state texts", mode, bacnet.PropertyRef{Property: bacnet.PropStateText, Index: at(4)}, bacnet.ErrInvalidArrayIndex},
	}
	for _, c := range cases {
		if got, err := d.Read(c.object, c.ref, 0); err == nil || *err != *c.want {
			t.Errorf("%s: %+v, %v; want %v", c.name, got, err, c.want)
		}
	}
}

func ids(results []bacnet.ReadResult) []bacnet.Property {
	var out []bacnet.Property
	for _, r := range results {
		out = append(out, r.Ref.Property)
	}
	return out
}

func TestTheSpecialPropertiesListWhatAnObjectHas(t *testing.T) {
	t.Parallel()
	d := bench(true)
	required := []bacnet.Property{bacnet.PropObjectIdentifier, bacnet.PropObjectName, bacnet.PropObjectType, bacnet.PropPresentValue,
		bacnet.PropStatusFlags, bacnet.PropEventState, bacnet.PropOutOfService, bacnet.PropUnits, bacnet.PropPriorityArray, bacnet.PropRelinquishDefault}
	optional := []bacnet.Property{bacnet.PropReliability, bacnet.PropMinPresValue, bacnet.PropMaxPresValue, bacnet.PropCOVIncrement}
	if got := ids(d.ReadSet(damper, bacnet.PropRequired, 0)); !reflect.DeepEqual(got, required) {
		t.Errorf("REQUIRED of an analog output = %v\nwant %v", got, required)
	}
	if got := ids(d.ReadSet(damper, bacnet.PropOptional, 0)); !reflect.DeepEqual(got, optional) {
		t.Errorf("OPTIONAL of an analog output = %v\nwant %v", got, optional)
	}
	all := d.ReadSet(damper, bacnet.PropAll, 0)
	if len(all) != len(required)+len(optional) {
		t.Errorf("ALL has %d properties, want %d", len(all), len(required)+len(optional))
	}
	for _, r := range all {
		if r.Err != nil || (len(r.Values) == 0) {
			t.Errorf("ALL: %v = %+v, %v", r.Ref.Property, r.Values, r.Err)
		}
		if r.Ref.Property == bacnet.PropPriorityArray && len(r.Values) != 16 {
			t.Errorf("ALL gives the priority array %d elements", len(r.Values))
		}
	}
	if got := d.ReadSet(device, bacnet.PropAll, 0); len(got) != 21 {
		t.Errorf("ALL of the device has %d properties, want 21: %v", len(got), ids(got))
	}
	missing := d.ReadSet(bacnet.ObjectID{Type: bacnet.BinaryInput, Instance: 99}, bacnet.PropAll, 0)
	if len(missing) != 1 || missing[0].Ref.Property != bacnet.PropAll || missing[0].Err == nil || *missing[0].Err != *bacnet.ErrUnknownObject {
		t.Errorf("ALL of an object the device has not = %+v, want one unknown-object error", missing)
	}
}

func TestObjectsAreFoundByIdentifierAndByName(t *testing.T) {
	t.Parallel()
	d := bench(false)
	if got, ok := d.FindByName("fan_cmd"); !ok || got != command {
		t.Errorf("fan_cmd = %v, %v", got, ok)
	}
	if got, ok := d.FindByName("unit-a"); !ok || got != device {
		t.Errorf("the device's own name = %v, %v", got, ok)
	}
	if _, ok := d.FindByName("absent"); ok {
		t.Error("a name no object has was found")
	}
	if d.Point(command) != "fan_cmd" || d.Point(device) != "" || d.Point(bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 9}) != "" {
		t.Error("Point does not name the model point of each object")
	}
	if inc, analog := d.COVIncrement(supply); inc != 0.5 || !analog {
		t.Errorf("COVIncrement(supply) = %v, %v", inc, analog)
	}
	if _, analog := d.COVIncrement(status); analog {
		t.Error("a binary input has a COV increment")
	}
}
