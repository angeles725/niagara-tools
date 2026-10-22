package retrocells

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseDeltasCell(t *testing.T) {
	t.Parallel()
	ds := func(n int) []Delta {
		out := []Delta{}
		for k := 1; k <= n; k++ {
			out = append(out, Delta{K: k})
		}
		return out
	}
	cases := []struct {
		name, cell string
		want       []Delta
		bad        bool
	}{
		{"zero deltas", "0", ds(0), false},
		{"one", "1", ds(1), false},
		{"three", "3", ds(3), false},
		{"leading zero is decimal, not octal", "010", ds(10), false},
		{"08 and 09 are decimal", "09", ds(9), false},
		{"all zeros", "000", ds(0), false},
		{"the maximum", "999", ds(MaxDeltas), false},
		{"over the maximum", "1000", nil, true},
		{"overflows int64", "18446744073709551617", nil, true},
		{"empty", "", nil, true},
		{"word", "x", nil, true},
		{"sign", "+1", nil, true},
		{"negative", "-1", nil, true},
		{"inner space (callers strip spaces first)", "1 2", nil, true},
		{"tab", "1\t", nil, true},
		{"non-ASCII digit", "١", nil, true},
	}
	for _, c := range cases {
		got, err := ParseDeltasCell(c.cell)
		if c.bad {
			var ce *CellError
			if err == nil || !errors.As(err, &ce) || ce.Cell != c.cell || got != nil {
				t.Errorf("%s: ParseDeltasCell(%q) = (%v, %v), want a *CellError for the cell", c.name, c.cell, got, err)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: ParseDeltasCell(%q) = (%v, %v), want (%v, nil)", c.name, c.cell, got, err, c.want)
		}
	}
}

func TestDeltaString(t *testing.T) {
	t.Parallel()
	if got := (Delta{K: 12}).String(); got != "Δ12" {
		t.Errorf("Delta{12}.String() = %q, want Δ12", got)
	}
}
