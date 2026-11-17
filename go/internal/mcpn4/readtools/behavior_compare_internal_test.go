package readtools

import (
	"reflect"
	"testing"
)

func TestCompareStatesSortsChangesMissingAndNewByPath(t *testing.T) {
	before := []pointState{{"/b", "21,5", "0"}, {"/a", "true", "0"}, {"/gone", "1", "0"}, {"/same", "7 °C", "0"}}
	after := []pointState{{"/a", "false", "2"}, {"/b", "21,6", "0"}, {"/same", "7 °C", "0"}, {"/new", "0", "0"}}
	d := compareStates(before, after, 0.05)
	if want := []pointChange{{"/a", "0", "2"}}; !reflect.DeepEqual(d.statusChanged, want) {
		t.Errorf("status %v", d.statusChanged)
	}
	if want := []pointChange{{"/a", "true", "false"}, {"/b", "21,5", "21,6"}}; !reflect.DeepEqual(d.valueChanged, want) {
		t.Errorf("value %v", d.valueChanged)
	}
	if !reflect.DeepEqual(d.missing, []string{"/gone"}) || !reflect.DeepEqual(d.added, []string{"/new"}) || d.unchanged != 1 {
		t.Errorf("missing %v added %v unchanged %d", d.missing, d.added, d.unchanged)
	}
}

func TestCompareStatesToleranceAppliesOnlyToNumbers(t *testing.T) {
	d := compareStates([]pointState{{"/n", "21.50", "0"}, {"/t", "on", "0"}}, []pointState{{"/n", "21.6", "0"}, {"/t", "On", "0"}}, 0.2)
	if len(d.valueChanged) != 1 || d.valueChanged[0].path != "/t" || d.unchanged != 1 {
		t.Fatalf("%+v", d)
	}
}

func TestNumberTextReadsLocaleDecimalsAndUnitsOnly(t *testing.T) {
	for in, want := range map[string]any{"21.5": 21.5, "21,5": 21.5, "-3 kW": -3.0, " 4,0 % ": 4.0, "1,234.5": nil, "true": nil, "NaN": nil, "1,2,3": nil} {
		got, ok := numberText(in)
		if want == nil && ok || want != nil && (!ok || got != want.(float64)) {
			t.Errorf("%q = %v %v, want %v", in, got, ok, want)
		}
	}
}
