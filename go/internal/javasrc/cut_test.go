package javasrc

import "testing"

// TestCutComments: every want was printed by the awk strip_comment() of toolbelt/lint-delays.sh.
func TestCutComments(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"a, b // c", "a, b"},
		{"a, b\t   // c", "a, b"},
		{"x /* y */ z", "x   z"},
		{"x /* y", "x"},
		{`s = "http://a" // c`, `s = "http://a"`},
		{`c = '/' + "/*" // d`, `c = '/' + "/*"`},
		{"/* don't */ k // g", "  k"},
		{`"\"//" + y // z`, `"\"//" + y`},
		{`'\'' // q`, `'\''`},
		{"a /* b */ /* c */ d", "a     d"},
		{"no comment  ", "no comment  "},
		{`"open // x`, `"open // x`},
		{"/", "/"},
		{"x/*/y*/z", "x z"},
		{"", ""},
	} {
		if got := CutComments(c.in); got != c.want {
			t.Errorf("CutComments(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
