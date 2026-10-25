package pyjson

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
)

// floatTexts are float() arguments around every branch of ParseFloat: whitespace, underscores,
// signs, exponents, the special words, Unicode decimal digits and what float() refuses.
var floatTexts = []string{"1", " 1.5\n", "1_000.25", "1__0", "_1", "1_", "1e5", "1E-05", ".5", "5.", "1e", "e5",
	"+3", "-0", "-0.0", "inf", "-Infinity", "INF", "nan", "-NaN", "+nan", "nanx", "1e400", "-1e400", "1e-400",
	"０.５", "١٢٣", "1 ", "　1　", "１_２", "0x10", "1.5.2", "", " ", "abc", "1,5", "Ⅻ", "²"}

func TestParseFloatDifferentialPython(t *testing.T) {
	in, _ := json.Marshal(floatTexts)
	out := golden.Open(t, "float").Run(`import json, sys
for s in json.load(sys.stdin):
    try:
        print(repr(float(s)))
    except ValueError:
        print("E")`, in)
	want := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	for i, s := range floatTexts {
		got := "E"
		if f, ok := ParseFloat(s); ok {
			got = FloatRepr(f)
			if math.IsNaN(f) {
				got = "nan"
			}
		}
		if got != want[i] {
			t.Errorf("float(%q): go %s, python %s", s, got, want[i])
		}
	}
}
