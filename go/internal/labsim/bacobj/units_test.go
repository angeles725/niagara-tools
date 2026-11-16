package bacobj

import "testing"

func TestUnitsAreNamedOrNumbered(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]uint32{"degrees-celsius": 62, "percent": 98, "no-units": 95, "kilowatt-hours": 19, "47": 47, "65535": 65535} {
		if got, ok := ParseUnits(text); !ok || got != want {
			t.Errorf("ParseUnits(%q) = %d, %v; want %d", text, got, ok, want)
		}
	}
	for _, text := range []string{"", "celsius", "-1", "65536", "6.2"} {
		if got, ok := ParseUnits(text); ok {
			t.Errorf("ParseUnits(%q) = %d, want a refusal", text, got)
		}
	}
	if UnitsName(62) != "degrees-celsius" || UnitsName(1234) != "1234" {
		t.Errorf("UnitsName: %q, %q", UnitsName(62), UnitsName(1234))
	}
	seen := map[uint32]string{}
	for _, name := range UnitNames() {
		n, _ := ParseUnits(name)
		if other, dup := seen[n]; dup {
			t.Errorf("%q and %q are both units %d", name, other, n)
		}
		seen[n] = name
	}
}
