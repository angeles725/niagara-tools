package jssrc

import (
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	toks := Tokens([]string{"f(a[1], {b: (c)}) ) (x"})
	m := Match(toks)
	for i, want := range map[int]string{1: ")", 3: "]", 7: "}", 10: ")"} {
		if got := m[i]; got < 0 || got >= len(toks) || toks[got].Text != want {
			t.Errorf("Match[%d] (%s) = %d, want its %s", i, toks[i].Text, got, want)
		}
	}
	if last := len(toks) - 2; m[last] != len(toks) || m[0] != -1 {
		t.Errorf("an unclosed ( maps to len(toks) and a name to -1: got %d, %d", m[last], m[0])
	}
}

func TestExpr(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"to the semicolon", "x = a + f(b, c); y", "a + f ( b , c )"},
		{"to a comma outside brackets", "x = [1, 2], y = 3", "[ 1 , 2 ]"},
		{"to an enclosing closer", "g(x = a ? b : c) + 1", "a ? b : c"},
		{"over lines while an operator continues", "x = a +\n b\n c()", "a + b"},
		{"ASI: a name on a new line after a name", "x = a\nb = 2", "a"},
		{"a member call continues on a new line", "x = a\n .b()", "a . b ( )"},
		{"to the end of input", "x = `t${1}`", "`t${1}`"},
	} {
		toks := Tokens(strings.Split(tc.src, "\n"))
		from := 0
		for toks[from].Text != "=" {
			from++
		}
		from++
		var got []string
		for _, tk := range toks[from:Expr(toks, from)] {
			s := tk.Text
			if tk.Kind == Template {
				s = "`" + s + "`"
			}
			got = append(got, s)
		}
		if g := strings.Join(got, " "); g != tc.want {
			t.Errorf("%s: Expr = %q, want %q", tc.name, g, tc.want)
		}
	}
}
