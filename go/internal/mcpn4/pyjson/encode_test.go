package pyjson

import (
	"errors"
	"math"
	"math/big"
	"strings"
	"testing"
)

// TestFloatRepr pins Python's repr(float) (the 'r' format): shortest round-trip digits, fixed
// notation while the decimal point sits in -4 < decpt <= 16, otherwise d.ddde+XX with at least two
// exponent digits; nan/inf spell as Python's str(float) does.
func TestFloatRepr(t *testing.T) {
	tenth := 0.1 // a variable: constant arithmetic would be exact
	tests := []struct {
		name string
		in   float64
		want string
	}{
		{"whole number keeps .0", 100.0, "100.0"},
		{"zero", 0, "0.0"},
		{"negative zero", math.Copysign(0, -1), "-0.0"},
		{"1e16 switches to exponent", 1e16, "1e+16"},
		{"1e15 stays fixed", 1e15, "1000000000000000.0"},
		{"largest fixed with fraction", 1234567890123456.7, "1234567890123456.8"},
		{"17 significant digits go exponent", 123456789012345678.0, "1.2345678901234568e+17"},
		{"0.0001 stays fixed", 0.0001, "0.0001"},
		{"1e-05 two-digit exponent", 1e-05, "1e-05"},
		{"1.5e-07", 1.5e-07, "1.5e-07"},
		{"three-digit exponent", 1e300, "1e+300"},
		{"max float", math.MaxFloat64, "1.7976931348623157e+308"},
		{"smallest subnormal", math.SmallestNonzeroFloat64, "5e-324"},
		{"shortest round trip", tenth + 2*tenth, "0.30000000000000004"},
		{"negative fixed", -2.5, "-2.5"},
		{"nan", math.NaN(), "nan"},
		{"inf", math.Inf(1), "inf"},
		{"-inf", math.Inf(-1), "-inf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FloatRepr(tt.in); got != tt.want {
				t.Fatalf("FloatRepr(%v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func obj(kv ...any) *Object {
	o := NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// TestMarshalBoundary pins json.dumps(obj) and json.dumps(obj, sort_keys=True,
// separators=(",", ":")) (safety.canonical) for the value shapes the server emits.
func TestMarshalBoundary(t *testing.T) {
	tests := []struct {
		name      string
		in        any
		dumps     string
		canonical string
	}{
		{"null", nil, "null", "null"},
		{"bools", []any{true, false}, "[true, false]", "[true,false]"},
		{"empty containers", []any{[]any{}, NewObject()}, "[[], {}]", "[[],{}]"},
		{"insertion order kept, sort on demand", obj("b", 1, "a", 2), `{"b": 1, "a": 2}`, `{"a":2,"b":1}`},
		{"reassign keeps first position", obj("x", 1, "y", 2, "x", 3), `{"x": 3, "y": 2}`, `{"x":3,"y":2}`},
		{"nested sort", obj("z", obj("b", nil, "a", []any{obj("d", 1, "c", 2)})),
			`{"z": {"b": null, "a": [{"d": 1, "c": 2}]}}`, `{"z":{"a":[{"c":2,"d":1}],"b":null}}`},
		{"floats", []any{100.0, 1e16, 1e15, 0.0001, 1e-05, math.Copysign(0, -1)},
			"[100.0, 1e+16, 1000000000000000.0, 0.0001, 1e-05, -0.0]", "[100.0,1e+16,1000000000000000.0,0.0001,1e-05,-0.0]"},
		{"non-finite floats", []any{math.NaN(), math.Inf(1), math.Inf(-1)},
			"[NaN, Infinity, -Infinity]", "[NaN,Infinity,-Infinity]"},
		{"go ints", []any{0, -7, int64(math.MaxInt64)}, "[0, -7, 9223372036854775807]", "[0,-7,9223372036854775807]"},
		{"int literal kept verbatim", IntLiteral("123456789012345678901234567890"),
			"123456789012345678901234567890", "123456789012345678901234567890"},
		{"float number", FloatNumber(2.5), "2.5", "2.5"},
		{"short escapes", "\"\\\n\r\t\b\f/", `"\"\\\n\r\t\b\f/"`, `"\"\\\n\r\t\b\f/"`},
		{"controls lowercase hex", "\x00\x1f\x7f", `"\u0000\u001f\u007f"`, `"\u0000\u001f\u007f"`},
		{"no html escaping", "<a href='x'>&</a>", `"<a href='x'>&</a>"`, `"<a href='x'>&</a>"`},
		{"bmp non-ascii", "\u00e9\u2028\uffff", `"\u00e9\u2028\uffff"`, `"\u00e9\u2028\uffff"`},
		{"surrogate pair", "\U0001f600", `"\ud83d\ude00"`, `"\ud83d\ude00"`},
		{"lone surrogate (wtf-8)", "a\xed\xa0\x80b", `"a\ud800b"`, `"a\ud800b"`},
		{"keys escaped and sorted by code point", obj("\u00e9", 1, "z", 2, "\U0001f600", 3, "\uffff", 4),
			`{"\u00e9": 1, "z": 2, "\ud83d\ude00": 3, "\uffff": 4}`, `{"z":2,"\u00e9":1,"\uffff":4,"\ud83d\ude00":3}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Marshal(tt.in, Options{})
			if err != nil || string(got) != tt.dumps {
				t.Fatalf("dumps = %q, %v; want %q", got, err, tt.dumps)
			}
			got, err = Marshal(tt.in, Canonical)
			if err != nil || string(got) != tt.canonical {
				t.Fatalf("canonical = %q, %v; want %q", got, err, tt.canonical)
			}
		})
	}
}

// TestMarshalRejectsUnsupported mirrors Python's TypeError for a non-JSON value: nothing is
// written, and the message names the Go type.
func TestMarshalRejectsUnsupported(t *testing.T) {
	_, err := Marshal([]any{1, struct{}{}}, Options{})
	if err == nil || !strings.Contains(err.Error(), "struct {}") {
		t.Fatalf("err = %v, want unsupported-type error", err)
	}
	if _, err := Marshal(map[string]any{"a": 1}, Options{}); err == nil {
		t.Fatal("map[string]any must be refused: its key order is random")
	}
}

// TestNumberFloat pins float(x) for a decoded number: an int literal converts with correct
// rounding, and one beyond the float range fails with Python's OverflowError text.
func TestNumberFloat(t *testing.T) {
	// max is int(sys.float_info.max); half an ulp (2^970) above it is where float() overflows.
	max, _ := new(big.Float).SetFloat64(math.MaxFloat64).Int(nil)
	halfULP := new(big.Int).Lsh(big.NewInt(1), 970)
	below := new(big.Int).Add(max, new(big.Int).Sub(halfULP, big.NewInt(1)))
	tie := new(big.Int).Add(max, halfULP)
	tests := []struct {
		name    string
		in      Number
		want    float64
		wantErr bool
	}{
		{"small int", IntLiteral("100"), 100, false},
		{"negative zero int is zero", IntLiteral("-0"), 0, false},
		{"2^53+1 rounds to even", IntLiteral("9007199254740993"), 9007199254740992, false},
		{"max float int", IntLiteral(max.String()), math.MaxFloat64, false},
		{"just under half ulp rounds down", IntLiteral(below.String()), math.MaxFloat64, false},
		{"half ulp tie rounds to overflow", IntLiteral(tie.String()), 0, true},
		{"1e400 written as an int", IntLiteral("1" + strings.Repeat("0", 400)), 0, true},
		{"negative overflow", IntLiteral("-1" + strings.Repeat("0", 400)), 0, true},
		{"float stays float", FloatNumber(1e300), 1e300, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Float64()
			if tt.wantErr {
				var oe *OverflowError
				if !errors.As(err, &oe) || err.Error() != "int too large to convert to float" {
					t.Fatalf("err = %v, want OverflowError", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("Float64() = %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}
