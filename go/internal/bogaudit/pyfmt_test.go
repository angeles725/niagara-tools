package bogaudit

import "testing"

func TestPyRepr(t *testing.T) {
	for _, c := range reprCases {
		if got := pyRepr(c.in); got != c.want {
			t.Errorf("pyRepr(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestPyFloatRepr(t *testing.T) {
	for _, c := range floatReprCases {
		if got := pyFloatRepr(c.in); got != c.want {
			t.Errorf("pyFloatRepr(%v) = %s, want %s", c.in, got, c.want)
		}
	}
}

// TestPyFloat pins float() acceptance; "٣" (Arabic-Indic three) is the declared ASCII-only difference.
func TestPyFloat(t *testing.T) {
	for _, c := range floatParseCases {
		f, ok := pyFloat(c.in)
		if c.in == "٣" {
			c.ok, c.want = false, ""
		}
		if ok != c.ok || (ok && pyFloatRepr(f) != c.want) {
			t.Errorf("pyFloat(%q) = %v %v, want %v %s", c.in, pyFloatRepr(f), ok, c.ok, c.want)
		}
	}
}

func TestPyInt(t *testing.T) {
	for in, want := range map[string]string{"0": "0", "000": "0", "02": "2", "10": "10"} {
		if got := pyInt(in); got != want {
			t.Errorf("pyInt(%q) = %s, want %s", in, got, want)
		}
	}
	if !intLess("9", "10") || intLess("10", "9") || intLess("3", "3") {
		t.Error("intLess orders by value")
	}
}
