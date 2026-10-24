package citeparse

import (
	"reflect"
	"testing"
)

func TestParseRanges(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want []Range
		ok   bool
	}{
		{"441-455", []Range{{441, 455}}, true},
		{"87-90,148-150, 152", []Range{{87, 90}, {148, 150}, {152, 152}}, true},
		{"12a", nil, false},
		{"", nil, false},
		{"3-", nil, false},
	}
	for _, tt := range cases {
		if got, ok := ParseRanges(tt.in); ok != tt.ok || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("ParseRanges(%q) = %v %v, want %v %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestLocator(t *testing.T) {
	t.Parallel()
	p := Pointer{Lines: []Range{{1, 2}, {5, 5}}, Sections: []string{"1.2"}, Rows: []Range{{3, 3}}, Claims: []string{"C1"},
		Rules: []string{"2"}, Delta: 4, Tokens: []string{"T1"}}
	if got, want := p.Locator(), ":1-2,5 §1.2 row 3 C1 rule 2 Δ4 T1"; got != want {
		t.Errorf("Locator() = %q, want %q", got, want)
	}
	if got := (Pointer{Rows: []Range{{1, 4}}}).Locator(); got != "rows 1-4" {
		t.Errorf("rows: %q", got)
	}
}
