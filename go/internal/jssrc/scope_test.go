package jssrc

import (
	"strings"
	"testing"
)

func TestBindings(t *testing.T) {
	src := []string{
		"const units = [1, 2], n = 3;",
		"function f(u, {a}, k = 2) { return u + a + k; }",
		"units.map(u => u * n);",
		"units.forEach((p, i) => { const q = p + i; }); q;",
		"for (const x of units.slice(1)) { x; } x;",
		"{ const n = 9; n; }",
		"n; t = `${n}`; let z;",
	}
	toks := Tokens(src)
	bs := Bindings(toks, Match(toks))
	text := func(r [2]int) string {
		var w []string
		for _, tk := range toks[r[0]:r[1]] {
			w = append(w, tk.Text)
		}
		return strings.Join(w, " ")
	}
	// nth returns the token index of the k-th (0-based) occurrence of name on 0-based line ln.
	nth := func(ln int, name string, k int) int {
		for i, tk := range toks {
			if tk.Line == ln && tk.Text == name && tk.Kind == Ident {
				if k == 0 {
					return i
				}
				k--
			}
		}
		t.Fatalf("no %s #%d on line %d", name, k, ln)
		return -1
	}
	for _, tc := range []struct {
		name  string
		ln, k int
		want  string // "<kind> <detail>" or "none"
	}{
		{"n", 2, 0, "decl const init=3"},
		{"u", 1, 1, "param 0 fn=function"},
		{"k", 1, 1, "param 2 fn=function"},
		{"a", 1, 1, "none"},
		{"u", 2, 1, "param 0 fn=u"},
		{"p", 3, 1, "param 0 fn=("},
		{"i", 3, 1, "param 1 fn=("},
		{"q", 3, 1, "none"},
		{"x", 4, 1, "of const init=units . slice ( 1 )"},
		{"x", 4, 2, "none"},
		{"n", 5, 1, "decl const init=9"},
		{"n", 6, 0, "decl const init=3"},
		{"z", 6, 0, "decl let init="},
	} {
		got := "none"
		if b := Lookup(bs, tc.name, nth(tc.ln, tc.name, tc.k)); b != nil {
			c := map[bool]string{true: "const", false: "let"}[b.Const]
			switch b.Kind {
			case Decl:
				got = "decl " + c + " init=" + text(b.Init)
			case Of:
				got = "of " + c + " init=" + text(b.Init)
			case Param:
				got = "param " + string(rune('0'+b.Index)) + " fn=" + toks[b.Fn].Text
			}
		}
		if got != tc.want {
			t.Errorf("%s at line %d #%d: got %q, want %q", tc.name, tc.ln, tc.k, got, tc.want)
		}
	}
}

func TestTemplateSubs(t *testing.T) {
	toks := Tokens([]string{"x = `a${ u.name }b${ `c${d}` }`;"})
	tpl := toks[2]
	if tpl.Kind != Template || len(tpl.Subs) != 2 || show(tpl.Subs[0]) != "0:ident:u 0:punct:. 0:ident:name" ||
		len(tpl.Subs[1]) != 1 || len(tpl.Subs[1][0].Subs) != 1 || show(tpl.Subs[1][0].Subs[0]) != "0:ident:d" {
		t.Errorf("template subs = %+v", tpl)
	}
	if show(toks) != "0:ident:x 0:punct:= 0:template:a${ u.name }b${ `c${d}` } 0:punct:;" {
		t.Errorf("stream = %s", show(toks))
	}
}
