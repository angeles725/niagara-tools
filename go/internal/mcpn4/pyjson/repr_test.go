package pyjson

import "testing"

// TestRepr pins repr(json.loads(in)) as python3 3.15 prints it.
func TestRepr(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, in, want string }{
		{"list and dict", `[1, 2.5, "a'b", {"k": [null, true, false]}]`,
			`[1, 2.5, "a'b", {'k': [None, True, False]}]`},
		{"escapes and printable non-ASCII", `"tab\there\u0000é \ud800😀 \u007f"`,
			"'tab\\there\\x00é\\xa0\\ud800\U0001F600\\u2028\\x7f'"},
		{"both quotes", `"both ' and \""`, `'both \' and "'`},
		{"float overflow literal", `1e400`, `inf`},
		{"negative zero", `-0.0`, `-0.0`},
		{"big int", `12345678901234567890123`, `12345678901234567890123`},
		{"non-finite and exponent", `{"a": NaN, "b": -Infinity, "c": 1E+16}`, `{'a': nan, 'b': -inf, 'c': 1e+16}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v, err := Unmarshal([]byte(tc.in))
			if err != nil {
				t.Fatalf("Unmarshal(%s): %v", tc.in, err)
			}
			if got := Repr(v); got != tc.want {
				t.Errorf("Repr = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestStrKeepsAStringBareAndReprsTheRest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		v    any
		want string
	}{{"n4_x", "n4_x"}, {nil, "None"}, {IntLiteral("5"), "5"}, {FloatNumber(1), "1.0"}, {[]any{"a"}, "['a']"}} {
		if got := Str(tc.v); got != tc.want {
			t.Errorf("Str(%#v) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestStrReprPicksDoubleQuotesOnlyForASingleQuoteAlone(t *testing.T) {
	t.Parallel()
	if got := StrRepr("it's"); got != `"it's"` {
		t.Errorf("StrRepr = %s", got)
	}
	if got := StrRepr(`say "hi"`); got != `'say "hi"'` {
		t.Errorf("StrRepr = %s", got)
	}
}
