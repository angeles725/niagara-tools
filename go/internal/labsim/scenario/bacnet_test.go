package scenario

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

const withBACnet = `{
  "devices": [
    {"name": "unit-a", "unit": 1,
     "bacnet": {"device_instance": 1001, "vendor_id": 555, "vendor_name": "Lab", "model": "bench", "max_apdu": 480, "cov": true},
     "points": [
      {"name": "run", "table": "coil", "address": 0, "value": 1, "writable": true,
       "bacnet": {"object_type": "binary-output", "instance": 1, "commandable": true, "active_text": "On", "inactive_text": "Off"}},
      {"name": "alarm", "table": "discrete_input", "address": 0, "bacnet": {"object_type": "binary-input", "instance": 1}},
      {"name": "setpoint", "table": "holding_register", "address": 0, "type": "float32", "value": 21.5, "writable": true,
       "bacnet": {"object_type": "analog-value", "instance": 1, "units": "degrees-celsius", "min_pres_value": 10, "max_pres_value": 30, "resolution": 0.5}},
      {"name": "supply", "table": "input_register", "address": 0, "type": "int16", "value": 180,
       "behaviour": {"kind": "sine", "amplitude": 20, "period_ms": 60000},
       "bacnet": {"object_type": "analog-input", "instance": 1, "units": "62", "cov_increment": 0.5, "description": "Supply air x10"}},
      {"name": "mode", "table": "holding_register", "address": 2, "type": "uint16", "value": 2, "writable": true,
       "bacnet": {"object_type": "multi-state-value", "instance": 1, "state_texts": ["Off", "Heat", "Cool"]}},
      {"name": "hidden", "table": "holding_register", "address": 3, "type": "uint16", "value": 7}
     ]},
    {"name": "unit-b", "unit": 2, "bacnet": {"device_instance": 1002, "listen": "127.0.0.1:47810"},
     "points": [{"name": "count", "table": "input_register", "address": 0, "bacnet": {"object_type": "analog-input", "instance": 1}}]}
  ]
}`

func parseBACnet(t *testing.T) *File {
	t.Helper()
	f, faults := Parse([]byte(withBACnet))
	if len(faults) != 0 {
		t.Fatalf("the BACnet scenario has faults: %v", faults)
	}
	return f
}

func TestParseFillsTheBACnetDefaults(t *testing.T) {
	t.Parallel()
	f := parseBACnet(t)
	a, b := f.Devices[0].BACnet, f.Devices[1].BACnet
	if a.Listen != DefaultBACnetListen || a.MaxAPDU != 480 || a.VendorName != "Lab" || a.Firmware == "" || a.Application == "" {
		t.Errorf("the first device: %+v", a)
	}
	if b.Listen != "127.0.0.1:47810" || b.MaxAPDU != 1476 || b.VendorID != DefaultVendorID || b.VendorName != DefaultVendorName || b.Model != DefaultModel || b.COV {
		t.Errorf("the second device: %+v", b)
	}
	if got := f.Devices[1].Points[0].BACnet.Units; got != "no-units" {
		t.Errorf("an analog object with no units has units %q, want no-units", got)
	}
	if got := f.Devices[0].Points[0].BACnet.Units; got != "" {
		t.Errorf("a binary object got units %q", got)
	}
}

func TestBuildLaysTheBACnetObjectsOverThePoints(t *testing.T) {
	t.Parallel()
	built := parseBACnet(t).Build()
	if len(built.BACnet) != 2 || built.BACnet[0].Listen != DefaultBACnetListen || built.BACnet[1].Listen != "127.0.0.1:47810" {
		t.Fatalf("Build().BACnet = %+v", built.BACnet)
	}
	d := built.BACnet[0].Device
	if d.Model() != built.Devices[0] {
		t.Error("the BACnet device does not share the points of the Modbus unit")
	}
	if spec := d.Spec(); spec.Instance != 1001 || spec.Name != "unit-a" || spec.VendorID != 555 || spec.Model != "bench" || spec.MaxAPDU != 480 || !spec.COV {
		t.Errorf("device spec %+v", spec)
	}
	want := []string{"device,1001", "binary-output,1", "binary-input,1", "analog-value,1", "analog-input,1", "multi-state-value,1"}
	got := d.Objects()
	if len(got) != len(want) {
		t.Fatalf("objects %v, want %v (a point with no mapping has no object)", got, want)
	}
	for i, id := range got {
		if id.String() != want[i] {
			t.Errorf("object %d = %v, want %s", i, id, want[i])
		}
	}
	read := func(object string, property bacnet.Property) string {
		kind, _ := bacnet.ParseObjectType(strings.Split(object, ",")[0])
		v, err := d.Read(bacnet.ObjectID{Type: kind, Instance: 1}, bacnet.PropertyRef{Property: property}, 0)
		if err != nil || len(v) == 0 {
			t.Fatalf("%s %v: %v", object, property, err)
		}
		return v[0].String()
	}
	for _, c := range []struct {
		object   string
		property bacnet.Property
		want     string
	}{
		{"binary-output,1", bacnet.PropPresentValue, "1"}, {"binary-output,1", bacnet.PropActiveText, "On"},
		{"binary-output,1", bacnet.PropRelinquishDefault, "1"},
		{"analog-value,1", bacnet.PropPresentValue, "21.5"}, {"analog-value,1", bacnet.PropUnits, "62"},
		{"analog-value,1", bacnet.PropMinPresValue, "10"}, {"analog-value,1", bacnet.PropMaxPresValue, "30"},
		{"analog-value,1", bacnet.PropResolution, "0.5"},
		{"analog-input,1", bacnet.PropUnits, "62"}, {"analog-input,1", bacnet.PropCOVIncrement, "0.5"},
		{"analog-input,1", bacnet.PropDescription, "Supply air x10"}, {"analog-input,1", bacnet.PropPresentValue, "180"},
		{"multi-state-value,1", bacnet.PropPresentValue, "2"}, {"multi-state-value,1", bacnet.PropNumberOfStates, "3"},
	} {
		if got := read(c.object, c.property); got != c.want {
			t.Errorf("%s %v = %s, want %s", c.object, c.property, got, c.want)
		}
	}
	real := func(f float32) []bacnet.Value { return []bacnet.Value{bacnet.Real(f)} }
	pv := bacnet.PropertyRef{Property: bacnet.PropPresentValue}
	// The scenario's writable makes a value that is not commandable writable;
	// an input is never written from BACnet, whatever the Modbus table allows.
	if err := d.Write(bacnet.WriteProperty{Object: bacnet.ObjectID{Type: bacnet.AnalogValue, Instance: 1}, Ref: pv, Values: real(25)}, 0); err != nil {
		t.Errorf("a write of the writable analog value: %v", err)
	}
	if err := d.Write(bacnet.WriteProperty{Object: bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 1}, Ref: pv, Values: real(1)}, 0); err == nil {
		t.Error("a write of an analog input was accepted")
	}
	if plain := parse(t).Build(); len(plain.BACnet) != 0 {
		t.Errorf("a scenario with no bacnet section built %d BACnet devices", len(plain.BACnet))
	}
}

func TestValidateNamesEachBACnetFaultOnce(t *testing.T) {
	t.Parallel()
	neg, low, big := -1, 40.0, 4194303
	cases := []struct {
		name   string
		break_ func(*File)
		want   string
	}{
		{"no device instance", func(f *File) { f.Devices[0].BACnet.DeviceInstance = nil }, "bacnet: no device_instance"},
		{"a negative device instance", func(f *File) { f.Devices[0].BACnet.DeviceInstance = &neg }, "device_instance -1 is outside 0-4194302"},
		{"the wildcard device instance", func(f *File) { f.Devices[0].BACnet.DeviceInstance = &big }, "device_instance 4194303 is outside 0-4194302"},
		{"a device instance used twice", func(f *File) { f.Devices[1].BACnet.DeviceInstance = f.Devices[0].BACnet.DeviceInstance }, `device_instance 1001 is already device "unit-a"`},
		{"a listen address without a port", func(f *File) { f.Devices[0].BACnet.Listen = "127.0.0.1" }, `bacnet: listen "127.0.0.1" is not an IPv4 address and a port`},
		{"an IPv6 listen address", func(f *File) { f.Devices[0].BACnet.Listen = "[::1]:47809" }, `listen "[::1]:47809" is not an IPv4 address and a port`},
		{"a listen name", func(f *File) { f.Devices[0].BACnet.Listen = "localhost:47809" }, `listen "localhost:47809" is not an IPv4 address and a port`},
		{"a listen address used twice", func(f *File) { f.Devices[1].BACnet.Listen = DefaultBACnetListen }, `listen "127.0.0.1:47809" is already device "unit-a"`},
		{"a vendor id of 17 bits", func(f *File) { f.Devices[0].BACnet.VendorID = 70000 }, "vendor_id 70000 is outside 0-65535"},
		{"a max APDU no device states", func(f *File) { f.Devices[0].BACnet.MaxAPDU = 1000 }, "max_apdu 1000 is none of 50, 128, 206, 480, 1024, 1476"},
		{"a mapped point in a device with no bacnet", func(f *File) { f.Devices[1].BACnet = nil }, "bacnet: the point is mapped but its device has no bacnet section"},
		{"an unknown object type", func(f *File) { f.Devices[0].Points[0].BACnet.ObjectType = "binary_output" }, `bacnet: object_type "binary_output" is none of analog-input`},
		{"the device as a point", func(f *File) { f.Devices[0].Points[0].BACnet.ObjectType = "device" }, `object_type "device" is none of`},
		{"no instance", func(f *File) { f.Devices[0].Points[0].BACnet.Instance = nil }, "bacnet: no instance"},
		{"an instance out of range", func(f *File) { f.Devices[0].Points[0].BACnet.Instance = &big }, "instance 4194303 is outside 0-4194302"},
		{"an object used twice", func(f *File) {
			f.Devices[0].Points[3].BACnet.ObjectType = "analog-value"
		}, `analog-value 1 is already point "setpoint"`},
		{"unknown units", func(f *File) { f.Devices[0].Points[2].BACnet.Units = "celsius" }, `units "celsius" is neither a unit name`},
		{"units on a binary object", func(f *File) { f.Devices[0].Points[0].BACnet.Units = "percent" }, "units belong to analog objects, not to binary-output"},
		{"a commandable input", func(f *File) { f.Devices[0].Points[1].BACnet.Commandable = true }, "an input (binary-input) cannot be commandable"},
		{"a multi-state with no states", func(f *File) { f.Devices[0].Points[4].BACnet.StateTexts = nil }, "multi-state-value needs state_texts"},
		{"states on an analog object", func(f *File) { f.Devices[0].Points[2].BACnet.StateTexts = []string{"a"} }, "state_texts belong to multi-state objects, not to analog-value"},
		{"an initial state past the last", func(f *File) { f.Devices[0].Points[4].Value = 4 }, "value 4 is not one of the 3 states (1-3)"},
		{"a binary text on an analog object", func(f *File) { f.Devices[0].Points[2].BACnet.ActiveText = "On" }, "active_text and inactive_text belong to binary objects, not to analog-value"},
		{"limits on a binary object", func(f *File) { f.Devices[0].Points[0].BACnet.Min = &low }, "min_pres_value, max_pres_value, resolution and cov_increment belong to analog objects, not to binary-output"},
		{"a minimum above the maximum", func(f *File) { f.Devices[0].Points[2].BACnet.Min = &low }, "min_pres_value 40 is above max_pres_value 30"},
		{"a negative COV increment", func(f *File) { f.Devices[0].Points[3].BACnet.COVIncrement = -1 }, "cov_increment -1 is negative"},
	}
	for _, c := range cases {
		f := parseBACnet(t)
		c.break_(f)
		faults := Validate(f)
		if len(faults) != 1 || !strings.Contains(faults[0], c.want) {
			t.Errorf("%s: faults = %q, want one holding %q", c.name, faults, c.want)
		}
	}
	// A fault names its device and its point, as the Modbus ones do.
	f := parseBACnet(t)
	f.Devices[0].Points[2].BACnet.Units = "celsius"
	f.Devices[1].BACnet.MaxAPDU = 7
	if faults := Validate(f); len(faults) != 2 || !strings.HasPrefix(faults[0], `device 1 "unit-a", point 3 "setpoint": bacnet: `) ||
		!strings.HasPrefix(faults[1], `device 2 "unit-b": bacnet: `) {
		t.Errorf("faults = %q", faults)
	}
}
