package pyjson

import (
	"strings"
	"testing"
)

// decodeCase is one json.loads input and the json.dumps text of its result, or "" for a
// ValueError. The differential test replays every case against python3 as well.
type decodeCase struct {
	name  string
	in    string
	dumps string
}

var decodeCases = []decodeCase{
	{"int", "1", "1"},
	{"negative zero int is int zero", "-0", "0"},
	{"negative zero float", "-0.0", "-0.0"},
	{"float keeps kind", "1.0", "1.0"},
	{"exponent makes a float", "1e2", "100.0"},
	{"upper exponent with sign", "1.5E+3", "1500.0"},
	{"float overflow is inf", "1E400", "Infinity"},
	{"float underflow is zero", "1e-400", "0.0"},
	{"big int verbatim", "123456789012345678901234567890", "123456789012345678901234567890"},
	{"int at the digit limit", strings.Repeat("7", 4300), strings.Repeat("7", 4300)},
	{"negative int at the digit limit", "-" + strings.Repeat("7", 4300), "-" + strings.Repeat("7", 4300)},
	{"int past the digit limit", strings.Repeat("7", 4301), ""},
	{"bool is not a number", "[true, false, null]", "[true, false, null]"},
	{"non-finite literals", "[NaN,Infinity,-Infinity]", "[NaN, Infinity, -Infinity]"},
	{"duplicate key: first position, last value", `{"a":1,"b":2,"a":3}`, `{"a": 3, "b": 2}`},
	{"json whitespace around", " \t\n\r{ \"a\" : [ 1 , 2 ] }\r\n", `{"a": [1, 2]}`},
	{"surrogate pair escape", `"\ud83d\ude00"`, `"\ud83d\ude00"`},
	{"upper-case hex escape", `"\u00E9\uD83D\uDE00"`, `"\u00e9\ud83d\ude00"`},
	{"lone high surrogate", `"\ud800x"`, `"\ud800x"`},
	{"high then non-low escape", `"\ud83d\u0041"`, `"\ud83dA"`},
	{"lone low surrogate", `"\udc00"`, `"\udc00"`},
	{"short escapes and solidus", `"\"\\\/\b\f\n\r\t"`, `"\"\\/\b\f\n\r\t"`},
	{"raw non-ascii", "\"\u00e9\U0001f600\"", `"\u00e9\ud83d\ude00"`},
	{"raw DEL is allowed", "\"\x7f\"", `"\u007f"`},
	{"invalid utf-8 byte is surrogate-escaped", "\"a\xffb\"", `"a\udcffb"`},
	{"empty", "", ""},
	{"only whitespace", " ", ""},
	{"leading zero", "01", ""},
	{"dot without fraction", "1.", ""},
	{"fraction without int", ".5", ""},
	{"plus sign", "+1", ""},
	{"exponent without digits", "1e", ""},
	{"bare minus", "-", ""},
	{"trailing comma in array", "[1,]", ""},
	{"trailing comma in object", `{"a":1,}`, ""},
	{"single quotes", "{'a':1}", ""},
	{"non-string key", "{1:2}", ""},
	{"raw control in string", "\"\x01\"", ""},
	{"raw newline in string", "\"a\nb\"", ""},
	{"unknown escape", `"\x41"`, ""},
	{"short unicode escape", `"\u12"`, ""},
	{"unterminated string", `"abc`, ""},
	{"lower nan", "nan", ""},
	{"negative NaN", "-NaN", ""},
	{"partial literal", "tru", ""},
	{"extra data", "[1] x", ""},
	{"two values", "1 2", ""},
	{"utf-8 bom", "\ufeff1", ""},
	{"form feed is not json whitespace", "\f1", ""},
	{"invalid byte outside a string", "\xff", ""},
}

func TestUnmarshalBoundary(t *testing.T) {
	for _, tt := range decodeCases {
		t.Run(tt.name, func(t *testing.T) {
			v, err := Unmarshal([]byte(tt.in))
			if tt.dumps == "" {
				if err == nil {
					t.Fatalf("Unmarshal(%q) = %#v, want an error", tt.in, v)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal(%q): %v", tt.in, err)
			}
			got, err := Marshal(v, Options{})
			if err != nil || string(got) != tt.dumps {
				t.Fatalf("dumps(loads(%q)) = %q, %v; want %q", tt.in, got, err, tt.dumps)
			}
		})
	}
}

// TestUnmarshalKinds pins the Go types json.loads results map to: a bool never becomes a
// Number, and 100 and 100.0 keep their int/float kinds.
func TestUnmarshalKinds(t *testing.T) {
	v, err := Unmarshal([]byte(`{"i": 100, "f": 100.0, "b": true, "s": "x", "n": null, "l": []}`))
	if err != nil {
		t.Fatal(err)
	}
	o := v.(*Object)
	if got := strings.Join(o.Keys(), ","); got != "i,f,b,s,n,l" {
		t.Fatalf("keys = %s", got)
	}
	i, _ := o.Get("i")
	f, _ := o.Get("f")
	b, _ := o.Get("b")
	if n, ok := i.(Number); !ok || n.IsFloat() || n.String() != "100" {
		t.Fatalf("i = %#v, want int Number", i)
	}
	if n, ok := f.(Number); !ok || !n.IsFloat() || n.String() != "100.0" {
		t.Fatalf("f = %#v, want float Number", f)
	}
	if _, ok := b.(bool); !ok {
		t.Fatalf("b = %#v, want bool", b)
	}
}

// TestUnmarshalErrorOffset checks a syntax error reports where decoding stopped.
func TestUnmarshalErrorOffset(t *testing.T) {
	_, err := Unmarshal([]byte(`[1, 2,]`))
	se, ok := err.(*SyntaxError)
	if !ok || se.Offset != 6 {
		t.Fatalf("err = %#v, want *SyntaxError at offset 6", err)
	}
}
