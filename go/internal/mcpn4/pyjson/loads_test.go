package pyjson

import (
	"bufio"
	"encoding/hex"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
)

// loadsCases pin the error texts the BOX client shows ("transport failure: <text>") for a reply
// that is not JSON or not UTF-8. The expected strings were printed by CPython 3.15.
var loadsCases = []struct{ name, in, want string }{
	{"html page", "<html>not json", "Expecting value: line 1 column 1 (char 0)"},
	{"empty body", "", "Expecting value: line 1 column 1 (char 0)"},
	{"only whitespace", " ", "Expecting value: line 1 column 2 (char 1)"},
	{"lone minus", "-x", "Expecting value: line 1 column 1 (char 0)"},
	{"array trailing comma", "[1,\n 2,\n ]", "Illegal trailing comma before end of array: line 2 column 3 (char 6)"},
	{"object trailing comma", `{"a":1,}`, "Illegal trailing comma before end of object: line 1 column 7 (char 6)"},
	{"object opening comma", `{,}`, "Expecting property name enclosed in double quotes: line 1 column 2 (char 1)"},
	{"array opening comma", `[,]`, "Expecting value: line 1 column 2 (char 1)"},
	{"missing colon", `{"a" 1}`, "Expecting ':' delimiter: line 1 column 6 (char 5)"},
	{"missing comma", `[1 2]`, "Expecting ',' delimiter: line 1 column 4 (char 3)"},
	{"unterminated", `"abc`, "Unterminated string starting at: line 1 column 1 (char 0)"},
	{"backslash at the end", `"a\`, "Unterminated string starting at: line 1 column 1 (char 0)"},
	{"bad escape", `"a\x"`, `Invalid \escape: line 1 column 3 (char 2)`},
	{"control character", "\"a\x01\"", "Invalid control character at: line 1 column 3 (char 2)"},
	{"short u escape", `"\u12"`, `Invalid \uXXXX escape: line 1 column 3 (char 2)`},
	{"bad low surrogate escape", `"\ud800\u12"`, `Invalid \uXXXX escape: line 1 column 9 (char 8)`},
	{"extra data", "[1]x", "Extra data: line 1 column 4 (char 3)"},
	{"position counts code points", "[\"\u00e9\U0001f600\" x]", "Expecting ',' delimiter: line 1 column 7 (char 6)"},
	{"byte order mark", "\ufeff1", "Unexpected UTF-8 BOM (decode using utf-8-sig): line 1 column 1 (char 0)"},
	{"int digit limit", strings.Repeat("1", 4301), "Exceeds the limit (4300 digits) for integer string " +
		"conversion: value has 4301 digits; use sys.set_int_max_str_digits() to increase the limit"},
	{"invalid start byte", "a\x80", "'utf-8' codec can't decode byte 0x80 in position 1: invalid start byte"},
	{"cut sequence", "ab\xf0\x9f\x98", "'utf-8' codec can't decode bytes in position 2-4: unexpected end of data"},
	{"bad continuation", "\xe2\x82A", "'utf-8' codec can't decode bytes in position 0-1: invalid continuation byte"},
	{"encoded surrogate", "\xed\xa0\x80", "'utf-8' codec can't decode byte 0xed in position 0: invalid continuation byte"},
}

func TestLoadsErrorText(t *testing.T) {
	for _, tt := range loadsCases {
		t.Run(tt.name, func(t *testing.T) {
			v, err := Loads([]byte(tt.in))
			if err == nil || err.Error() != tt.want {
				t.Fatalf("Loads(%q) = %v, %v; want error %q", tt.in, v, err, tt.want)
			}
		})
	}
}

func TestDecodeUTF8ReplaceCountsMaximalSubparts(t *testing.T) {
	for in, want := range map[string]string{
		"\xe2\x82A": "\ufffdA", "\xed\xa0\x80": "\ufffd\ufffd\ufffd", "\xf0\x90\x80": "\ufffd",
		"\xc0\xaf": "\ufffd\ufffd", "ok\u00e9": "ok\u00e9", "\xf4\x90\x80\x80": "\ufffd\ufffd\ufffd\ufffd",
	} {
		if got, err := DecodeUTF8([]byte(in), true); err != nil || got != want {
			t.Errorf("DecodeUTF8(%q, replace) = %q, %v; want %q", in, got, err, want)
		}
	}
}

// loadsDriver prints, per input, json.loads(raw.decode()) re-dumped, or the ValueError text,
// then raw.decode("utf-8", "replace") as ASCII.
const loadsDriver = `
import json, sys
for line in sys.stdin:
    raw = bytes.fromhex(line.strip())
    try:
        out = "ok " + json.dumps(json.loads(raw.decode()))
    except ValueError as e:
        out = "err " + str(e)
    print(out + "\t" + json.dumps(raw.decode("utf-8", "replace")))
`

// TestDifferentialLoads runs the boundary cases, the decode corpus and random byte strings
// through CPython and through Loads / DecodeUTF8: the lines must be equal.
// Python's lines are the golden testdata/loads.golden.jsonl.
func TestDifferentialLoads(t *testing.T) {
	var docs []string
	for _, c := range loadsCases {
		docs = append(docs, c.in)
	}
	for _, c := range decodeCases {
		docs = append(docs, c.in)
	}
	rng := rand.New(rand.NewPCG(fuzzSeed, 7))
	g := gen{rng}
	for range fuzzDocs {
		docs = append(docs, g.doc())
	}
	alphabet := []byte("\x00\x7f\x80\xbf\xc0\xc2\xdf\xe0\xa0\xed\x9f\xef\xf0\x90\xf4\x8f\xf5\xff\"\\u[]{},: 1a")
	for range 2000 {
		b := make([]byte, 1+rng.IntN(8))
		for i := range b {
			b[i] = alphabet[rng.IntN(len(alphabet))]
		}
		docs = append(docs, string(b))
	}
	var in strings.Builder
	for _, d := range docs {
		in.WriteString(hex.EncodeToString([]byte(d)) + "\n")
	}
	out := golden.Open(t, "loads").Run(loadsDriver, []byte(in.String()))
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<20)
	i, fails := 0, 0
	for ; sc.Scan(); i++ {
		if got := loadsLine([]byte(docs[i])); got != sc.Text() && fails < 10 {
			fails++
			t.Errorf("doc %q:\n go  %s\n py  %s", docs[i], got, sc.Text())
		}
	}
	if i != len(docs) {
		t.Fatalf("python3 answered %d of %d docs", i, len(docs))
	}
}

func loadsLine(raw []byte) string {
	var out string
	if v, err := Loads(raw); err != nil {
		out = "err " + err.Error()
	} else {
		dumps, _ := Marshal(v, Options{})
		out = "ok " + string(dumps)
	}
	text, _ := DecodeUTF8(raw, true)
	quoted, _ := Marshal(text, Options{})
	return out + "\t" + string(quoted)
}
