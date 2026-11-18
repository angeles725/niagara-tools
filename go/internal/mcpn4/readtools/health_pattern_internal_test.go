package readtools

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

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
		got := downPattern(c.total, c.down, c.bad, true)
		if c.want == "" {
			if got != nil {
				t.Errorf("%s: pattern = %v, want null", c.name, got)
			}
			continue
		}
		p, ok := got.(*pyjson.Object)
		if !ok || p == nil || lookup(p, "kind") != c.want || lookup(p, "down") != c.down ||
			lookup(p, "total") != c.total || lookup(p, "hint") == "" {
			t.Errorf("%s: pattern = %v, want kind %s", c.name, got, c.want)
		}
	}
}

func TestDownPatternHintNamesWiringOnlyOnASerialNetwork(t *testing.T) {
	serial, _ := downPattern(4, 4, 4, true).(*pyjson.Object)
	ip, _ := downPattern(4, 4, 4, false).(*pyjson.Object)
	if lookup(serial, "hint") != hintBus || lookup(ip, "hint") != hintNetwork || lookup(ip, "kind") != patternBus {
		t.Errorf("serial %v, ip %v", serial, ip)
	}
}
