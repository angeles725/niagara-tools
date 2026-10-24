package retro

import "testing"

// The date half of fromisoformat on the forms CPython 3.15 reads (datetime.fromisoformat gave
// each expected date; TestParseTimestampDifferentialPython covers the whole parse).
func TestISODateForms(t *testing.T) {
	for _, c := range []struct {
		in      string
		sep     int
		y, m, d int
		ok      bool
	}{
		{"2026-10-01", 10, 2026, 10, 1, true}, {"20261001", 8, 2026, 10, 1, true},
		{"2026-W40", 8, 2026, 9, 28, true}, {"2026W40", 7, 2026, 9, 28, true},
		{"2026-W40-4", 10, 2026, 10, 1, true}, {"2026W404", 8, 2026, 10, 1, true},
		{"2020-W53-7", 10, 2021, 1, 3, true}, {"2015-W53-7", 10, 2016, 1, 3, true},
		{"2026-W53-1", 10, 2026, 12, 28, true}, {"9999-W52-5", 10, 9999, 12, 31, true},
		{"0001-W01-1", 10, 1, 1, 1, true}, {"2026-W00-1", 10, 0, 0, 0, false}, {"2026-W40-8", 10, 0, 0, 0, false},
		{"2027-W53-1", 10, 0, 0, 0, false}, {"0000-W01-1", 10, 0, 0, 0, false},
		{"2026-W40-4T10", 10, 2026, 10, 1, true}, {"2026-W40-1030", 8, 2026, 9, 28, true},
		{"2026W4041030", 7, 2026, 9, 28, true}, {"2026W40T10", 7, 2026, 9, 28, true}, {"2026-W40-", -1, 0, 0, 0, false},
		{"2026-10X01", 10, 0, 0, 0, false}, {"2026-1001", 10, 0, 0, 0, false}, {"202610-01", 8, 0, 0, 0, false},
	} {
		b, ok := sanitize(c.in)
		if !ok {
			t.Fatalf("%q does not sanitize", c.in)
		}
		sep := b.separator()
		if sep != c.sep {
			t.Errorf("%q: separator %d, want %d", c.in, sep, c.sep)
			continue
		}
		if sep < 0 {
			continue
		}
		if y, m, d, ok := b.date(sep); ok != c.ok || ok && (y != c.y || m != c.m || d != c.d) {
			t.Errorf("%q: %d-%d-%d %v", c.in, y, m, d, ok)
		}
	}
}

func TestSanitizeAllowsOneSurrogateSeparatorOnly(t *testing.T) {
	for in, want := range map[string]string{
		"2026-10-01\xed\xa0\x8010": "2026-10-01T10", "2026W40\xed\xa0\x8010": "2026W40T10",
		"2026-10-01T\xed\xa0\x80": "", "2026-1": "", "2026-10-01\xed\xa0\x80\xed\xa0\x80": "",
		"2026-10-01é10": "2026-10-01é10",
	} {
		b, ok := sanitize(in)
		if got := string(b); ok != (want != "") || got != want {
			t.Errorf("%q: %q %v", in, got, ok)
		}
	}
}
