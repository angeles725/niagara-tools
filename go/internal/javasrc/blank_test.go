package javasrc

import (
	"reflect"
	"testing"
)

func TestBlankLiterals(t *testing.T) {
	cases := []struct {
		name   string
		in     []string
		want   []string
		unterm bool
	}{
		{"empty line", []string{``}, []string{``}, false},
		{"no literal is returned unchanged", []string{`a(); { b; }`}, []string{`a(); { b; }`}, false},
		{"string contents removed, quotes kept", []string{`a("x;{}") b`}, []string{`a("") b`}, false},
		{"escaped quote stays inside", []string{`s = "a\"b"; c();`}, []string{`s = ""; c();`}, false},
		{"escaped backslash ends before the quote", []string{`s = "a\\" + x;`}, []string{`s = "" + x;`}, false},
		{"char literals", []string{`q = '"'; a = '\''; b = '{';`}, []string{`q = ''; a = ''; b = '';`}, false},
		{"unterminated literal ends at its line end", []string{`foo("abc {`, `bar();`}, []string{`foo("`, `bar();`}, true},
		{"trailing backslash in an open literal", []string{`foo("ab\`}, []string{`foo("`}, true},
		{"unterminated char literal", []string{`c = '{`}, []string{`c = '`}, true},
		{"text block across lines", []string{`s = """`, `  {"x"} ;`, `  end""";`, `x();`},
			[]string{`s = """`, ``, `""";`, `x();`}, false},
		{"escaped triple quote does not close a text block", []string{`"""`, `a\"""b`, `"""`},
			[]string{`"""`, ``, `"""`}, false},
		{"one-line text block", []string{`"""abc"""x`}, []string{`""""""x`}, false},
		{"an open text block is not an unterminated literal", []string{`s = """`, `  abc`}, []string{`s = """`, ``}, false},
	}
	for _, c := range cases {
		got, unterm := BlankLiterals(c.in, nil)
		if !reflect.DeepEqual(got, c.want) || unterm != c.unterm {
			t.Errorf("%s: BlankLiterals(%q) = %q, %v; want %q, %v", c.name, c.in, got, unterm, c.want, c.unterm)
		}
	}
}

func TestBlankLiteralsReportsClosedLiterals(t *testing.T) {
	in := []string{
		`f("bql:x", 'c', "y\"z");`,
		`g("open`,
		`s = """`,
		`"not a literal" """; h("k");`,
	}
	var got []Literal
	BlankLiterals(in, func(l Literal) { got = append(got, l) })
	want := []Literal{
		{Line: 0, Col: 3, Quote: '"', Text: `bql:x`},
		{Line: 0, Col: 7, Quote: '\'', Text: `c`},
		{Line: 0, Col: 11, Quote: '"', Text: `y\"z`},
		{Line: 3, Col: 8, Quote: '"', Text: `k`},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("literals = %+v\nwant       %+v", got, want)
	}
}

// BlankLine is the awk blank_str(): one line, no text-block state, a """ is three quotes.
func TestBlankLine(t *testing.T) {
	for in, want := range map[string]string{
		`log("a.cancel()"); b.cancel();`: `log(""); b.cancel();`,
		`c = '\''; d = "x\"y";`:          `c = ''; d = "";`,
		`no literals`:                    `no literals`,
		``:                               ``,
		`s = "a\\" + x;`:                 `s = "" + x;`,
		`q = '"';`:                       `q = '';`,
		`foo("abc {`:                     `foo("`,
		`foo("ab\`:                       `foo("`,
		`x = """`:                        `x = """`,
		`  {"x"} ;`:                      `  {""} ;`,
		`"""abc"""`:                      `""""""`,
		`a "b""c" d`:                     `a """" d`,
	} {
		if got := BlankLine(in); got != want {
			t.Errorf("BlankLine(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMaskLiterals: the bytes BlankLiterals removes are blanked in place, so every line keeps its length.
func TestMaskLiterals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		in, want []string
	}{
		{"no literal", []string{`a(); { b; }`}, []string{`a(); { b; }`}},
		{"string", []string{`a("x;{}") b`}, []string{`a("    ") b`}},
		{"escapes", []string{`s = "a\"b" + 'c' + '\''; c();`}, []string{`s = "    " + ' ' + '  '; c();`}},
		{"open at line end", []string{`foo("ab\`, `bar();`}, []string{`foo("   `, `bar();`}},
		{"text block", []string{`s = """`, `  add(x);`, `  end""";`, `x();`},
			[]string{`s = """`, `         `, `     """;`, `x();`}},
	} {
		got := MaskLiterals(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
		for i := range got {
			if len(got[i]) != len(tc.in[i]) {
				t.Errorf("%s: line %d length %d, want %d", tc.name, i, len(got[i]), len(tc.in[i]))
			}
		}
	}
}

func TestMaskLine(t *testing.T) {
	for in, want := range map[string]string{
		`f("a,b", c)`:     `f("   ", c)`,
		`x = 'q'; y`:      `x = ' '; y`,
		`s = "a\"b(" + t`: `s = "     " + t`,
		`open "abc`:       `open "   `,
		"plain":           "plain",
	} {
		got := MaskLine(in)
		if got != want || len(got) != len(in) {
			t.Errorf("MaskLine(%q) = %q, want %q", in, got, want)
		}
	}
}
