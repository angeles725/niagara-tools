package rcscan

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

func TestLiteral(t *testing.T) {
	t.Parallel()
	parse := func(src string) *lit {
		ts := jssrc.Tokens([]string{src})
		v, n := parseLit(ts, jssrc.Match(ts), 0)
		if n != len(ts) {
			t.Fatalf("parseLit(%q) stopped at %d of %d", src, n, len(ts))
		}
		return v
	}
	for _, tc := range []struct {
		src, path, write string
		want             bool
	}{
		{`"a"`, "", "", true},
		{`[1, -2, 'x', true, null, undefined, ` + "`t`" + `]`, "", "", true},
		{"[`a${x}`]", "", "", false},
		{`{ a: "x", b: c }`, "a", "", true},
		{`{ a: "x", b: c }`, "b", "", false},
		{`{ a: "x", b: c }`, "", "", false},
		{`{ a: "x", b: c }`, "z", "", false},
		{`{ a: 1 + 2 }`, "a", "", false},
		{`{ "k": { u: '°C' }, ...o }`, "k.u", "", true},
		{`{ n: "s" }`, "n.length", "", true},
		{`{ a: [1, 2] }`, "a.join", "", true},
		{`{ a: "x", b: "y" }`, "a", "b", true},
		{`{ a: "x", b: "y" }`, "a", "a", false},
		{`{ a: "x" }`, "a", "*", false},
		{`[{ a: "x" }]`, "", "*.b", true},
		{`[{ a: "x" }]`, "", "*.a", false},
		{`[{ a: "x" }]`, "", "length", false},
	} {
		v := parse(tc.src)
		if tc.write != "" {
			v.mark(strings.Split(tc.write, "."))
		}
		var path []string
		if tc.path != "" {
			path = strings.Split(tc.path, ".")
		}
		if got := v.holds(path); got != tc.want {
			t.Errorf("%s after write %q: holds(%q) = %v, want %v", tc.src, tc.write, tc.path, got, tc.want)
		}
	}
}
