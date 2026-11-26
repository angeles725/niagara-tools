package vendored

import "testing"

// TestCompare: three.js r-numbers compare as integers (r99 < r161), semver by its numbers (1.10.0 > 1.9.0, a
// missing part is 0, a leading v is ignored) and a release is above its pre-release; an r-number against a
// semver, or a text that is neither, is not comparable.
func TestCompare(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"r160", "r161", -1, true},
		{"r99", "r161", -1, true},
		{"R185", "r161", 1, true},
		{"r161", "r161", 0, true},
		{"1.10.0", "1.9.0", 1, true},
		{"v1.0", "1.0.0", 0, true},
		{"2", "1.9.9", 1, true},
		{"1.4.2-rc.1", "1.4.2", -1, true},
		{"1.4.2-rc.2", "1.4.2-rc.10", 1, true}, // pre-release tags compare as plain text
		{"r160", "0.160.0", 0, false},
		{"latest", "1.0.0", 0, false},
		{"1.0.0.0", "1.0.0", 0, false},
		{"r", "r161", 0, false},
	} {
		a, aok := parseVersion(c.a)
		b, bok := parseVersion(c.b)
		if !bok {
			t.Fatalf("%q does not parse", c.b)
		}
		if !aok {
			if c.ok {
				t.Errorf("%q does not parse", c.a)
			}
			continue
		}
		got, ok := a.compare(b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("compare(%q, %q) = %d, %v; want %d, %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

// TestConstraints: comma-joined comparisons must all hold; anything else is a parse error.
func TestConstraints(t *testing.T) {
	cs, err := parseAffected(">= 1.0.0, <1.4.2")
	if err != nil || len(cs) != 2 {
		t.Fatalf("parseAffected: %v, %v", cs, err)
	}
	for v, want := range map[string]bool{"0.9.9": false, "1.0.0": true, "1.4.1": true, "1.4.2": false} {
		got, ok := affects(cs, mustVersion(t, v))
		if !ok || got != want {
			t.Errorf("affects(%q) = %v, %v; want %v", v, got, ok, want)
		}
	}
	if _, ok := affects(cs, mustVersion(t, "r160")); ok {
		t.Error("an r-number against a semver range must not be comparable")
	}
	for _, bad := range []string{"", "r161", "older than r161", "< ", "<< r1", "< r161,"} {
		if _, err := parseAffected(bad); err == nil {
			t.Errorf("parseAffected(%q) must fail", bad)
		}
	}
	if cs, err := parseAffected("= r160"); err != nil || len(cs) != 1 {
		t.Errorf("= r160: %v, %v", cs, err)
	}
}

func mustVersion(t *testing.T, s string) version {
	t.Helper()
	v, ok := parseVersion(s)
	if !ok {
		t.Fatalf("%q does not parse", s)
	}
	return v
}
