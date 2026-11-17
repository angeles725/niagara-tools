package readtools

import "testing"

func TestDownPatternNamesTheBusOrTheSingleDevice(t *testing.T) {
	cases := []struct {
		name             string
		total, down, bad int
		want             string // "" is no pattern
	}{
		{"3 of 4 down: all but one", 4, 3, 3, patternBus},
		{"4 of 4 down", 4, 4, 4, patternBus},
		{"2 of 2 down", 2, 2, 2, patternBus},
		{"14 of 15 down", 15, 14, 14, patternBus},
		{"2 of 4 down: two still answer", 4, 2, 2, ""},
		{"1 of 3 down, the others ok", 3, 1, 1, patternSingle},
		{"1 of 2 down, the other ok: one down device says nothing of the bus", 2, 1, 1, patternSingle},
		{"1 of 3 down, another one faulted", 3, 1, 2, ""},
		{"a lone device down", 1, 1, 1, ""},
		{"nothing down", 3, 0, 1, ""},
		{"no devices", 0, 0, 0, ""},
	}
	for _, c := range cases {
		got := downPattern(c.total, c.down, c.bad)
		if c.want == "" {
			if got != nil {
				t.Errorf("%s: pattern = %v, want null", c.name, got)
			}
			continue
		}
		if got == nil || lookup(got, "kind") != c.want || lookup(got, "down") != c.down ||
			lookup(got, "total") != c.total || lookup(got, "hint") == "" {
			t.Errorf("%s: pattern = %v, want kind %s", c.name, got, c.want)
		}
	}
}
