package verifymodule

import (
	"reflect"
	"testing"
)

// TestGrepText pins the GNU grep 3.11 output rules under C.UTF-8 the checks read through: a line with an
// encoding error matches and counts but is never printed (-o prints a valid match from it), and a file
// holding a NUL byte prints nothing at all.
func TestGrepText(t *testing.T) {
	re := rx(`station:[a-z]`)
	latin := newGrepText("a \"station:x\"\nb \xe9 \"station:y\"\nc \"station:z\"\n")
	if got := latin.numbered(re); !reflect.DeepEqual(got, []string{`1:a "station:x"`, `3:c "station:z"`}) {
		t.Errorf("numbered %q", got)
	}
	if got := latin.only(re); !reflect.DeepEqual(got, []string{"station:x", "station:y", "station:z"}) {
		t.Errorf("only %q", got)
	}
	nul := newGrepText("a station:x\n\x00b\nc station:z")
	if nul.count(re) != 2 || nul.numbered(re) != nil || nul.only(re) != nil {
		t.Errorf("binary file: count %d numbered %q only %q", nul.count(re), nul.numbered(re), nul.only(re))
	}
	if newGrepText("").lines != nil || len(newGrepText("\n").lines) != 1 {
		t.Error("an empty input has no line, a lone newline one")
	}
}

// TestBashInt pins the operand rules of bash 5.2 arithmetic the coverage and --plano blocks used.
func TestBashInt(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int64
		err  string
	}{
		{"52", 52, ""}, {"010", 8, ""}, {"0x1F", 31, ""}, {"-3", -3, ""}, {"", 0, ""},
		{"99999999999999999999", 7766279631452241919, ""},
		{"08", 0, `08: value too great for base (error token is "08")`},
		{"2.5", 0, `2.5: syntax error: invalid arithmetic operator (error token is ".5")`},
		{"abc", 0, "abc: unbound variable"},
	} {
		got, err := bashInt(tc.in)
		msg := ""
		if err != nil {
			msg = err.msg
		}
		if got != tc.want || msg != tc.err {
			t.Errorf("bashInt(%q) = %d, %q; want %d, %q", tc.in, got, msg, tc.want, tc.err)
		}
	}
}
