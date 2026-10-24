package retro

import (
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// runPython runs script with mcp_n4 importable, the JSON of input on stdin, and decodes the
// JSON it prints.
func runPython(t *testing.T, script string, input any) []any {
	t.Helper()
	py := pytarget.Python(t)
	kit, _ := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit"))
	data, err := pyjson.Marshal(input, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(py, "-I", "-c", "import sys; sys.path.insert(0, sys.argv[1]); exec(sys.argv[2])", kit, script)
	cmd.Stdin = strings.NewReader(string(data))
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python: %v\n%s", err, stderr.String())
	}
	v, err := pyjson.Loads(out)
	if err != nil {
		t.Fatal(err)
	}
	return v.([]any)
}

const parseScript = `import datetime, json, sys
from mcp_n4 import retro
EPOCH = datetime.datetime(1970, 1, 1, tzinfo=datetime.timezone.utc)
def run(s):
    try:
        return (retro._parse_ts(s) - EPOCH) // datetime.timedelta(microseconds=1)
    except ValueError:
        return None
print(json.dumps([run(s) for s in json.load(sys.stdin)]))`

// timestampSeeds reach every branch of fromisoformat: each date form, the separator rules,
// fractions, offsets, hour 24, embedded NUL, surrogates, non-ASCII separators and the bounds.
var timestampSeeds = []string{
	"2026-10-01T10:00:00+00:00", "2026-10-01T10:00:00Z", "2026-10-01", "20261001", "2026-W40",
	"2026W40", "2026-W40-4", "2026W404", "2026-W40-4T10:00", "2026W404T1000", "2026-W40T10",
	"2026W40T10", "2026-W53-1", "2020-W53-7", "2015-W53-7", "2004-W53-6", "0001-W01-1", "9999-W52-5",
	"9999-W52-7", "2026-W00-1", "2026-W40-0", "2026-W40-8", "2026-W40-", "2026-W40X4", "2026-W4012",
	"2026-W40-10", "2026-W40-4-10", "2026W4041030", "2026W40410300", "2026-10-01 10:00",
	"2026-10-01é10:00", "2026-10-01\u3000T10:00", "2026-10-01\U0001F600 10:00", "2026-10-01\xed\xa0\x80T10",
	"2026-10\xed\xa0\x80\x30\x31", "2026-10-0\xed\xa0\x80T10", "2026-10-01T10:00\xed\xa0\x80",
	"2026-10-01T10", "2026-10-01T1000", "2026-10-01T100000", "2026-10-01T10:00", "2026-10-01T10:00:00",
	"2026-10-01T10:00:00.1", "2026-10-01T10:00:00.123456", "2026-10-01T10:00:00.1234567",
	"2026-10-01T10:00:00,5", "2026-10-01T10:00:00.", "2026-10-01T10.5", "2026-10-01T10:00.5",
	"2026-10-01T100000123", "2026-10-01T10:0000", "2026-10-01T1000:00", "2026-10-01T10:00:00:00",
	"2026-10-01T24:00", "2026-10-01T24:00:01", "2026-12-31T24:00:00", "9999-12-31T24:00", "0000-12-31T24:00",
	"2026-01-00T24:00", "2026-02-29T24:00", "2024-02-29T24:00", "2026-13-01T24:00", "2026-10-01T25:00",
	"2026-10-01T10:00:00+01", "2026-10-01T10:00:00+0130", "2026-10-01T10:00:00-01:30", "2026-10-01T10:00:00+013045",
	"2026-10-01T10:00:00+01:30:45.5", "2026-10-01T10:00:00+01:30:45.123456789", "2026-10-01T10:00:00+24:00",
	"2026-10-01T10:00:00+23:59:59.999999", "2026-10-01T10:00:00+1", "2026-10-01T10:00:00+123", "2026-10-01T10:00:00+",
	"2026-10-01T10:00:00+00:00x", "2026-10-01T10+05", "2026-10-01T10-05:00", "2026-10-01T10:00Zjunk",
	"2026-10-01T10:00:00Z", "2026-10-01T\x0010:00", "2026-10-01T10:00\x00", "2026-10-01T10:00:00.5\x00x",
	"2026-10-01T10:00+00:00\x00", "0001-01-01T00:00:00+01:00", "9999-12-31T23:59:59-23:59", "0000-01-01",
	"2026-02-29", "2024-02-29", "1900-02-29", "2000-02-29", "2026-04-31", "2026-1001", "202610-01",
	"2026-10-01T", "2026-10-01TT", " 2026-10-01T10:00 ", "\u20282026-10-01\u3000", "２０２６-10-01",
	"2026-10-01T１０:00", "nope", "", "short", "1234567", "2026W40", "2026-W4", "2026-W401",
	"20261001T1000+0100", "20261001T10:00", "2026-10-01T10:0", "ZZZZZZZZ", "2026-10-01Z", "2026-W40Z",
}

// mutants returns n strings made by a few edits of the seeds, from an alphabet of the bytes
// the parser treats specially.
func mutants(n int) []string {
	alphabet := []string{"0", "1", "2", "4", "9", "-", ":", "T", "W", "Z", "+", ".", ",", " ", "\x00", "é", "\u3000",
		"\xed\xa0\x80", "5", "3", "6"}
	r := rand.New(rand.NewSource(9))
	out := make([]string, n)
	for i := range out {
		seed, s := timestampSeeds[r.Intn(len(timestampSeeds))], []string{}
		for len(seed) > 0 { // by code point: a WTF-8 surrogate stays one
			_, size := pyjson.DecodeRune(seed)
			s, seed = append(s, seed[:size]), seed[size:]
		}
		for e := r.Intn(3) + 1; e > 0; e-- {
			pos := r.Intn(len(s) + 1)
			ch := alphabet[r.Intn(len(alphabet))]
			switch op := r.Intn(3); {
			case op == 0 || len(s) == 0 || pos == len(s):
				s = append(s[:pos], append([]string{ch}, s[pos:]...)...)
			case op == 1:
				s[pos] = ch
			default:
				s = append(s[:pos], s[pos+1:]...)
			}
		}
		out[i] = strings.Join(s, "")
	}
	return out
}

func TestParseTimestampDifferentialPython(t *testing.T) {
	corpus := append(append([]string(nil), timestampSeeds...), mutants(40000)...)
	input := make([]any, len(corpus))
	for i, s := range corpus {
		input[i] = s
	}
	want := runPython(t, parseScript, input)
	parsed := 0
	for i, s := range corpus {
		got, ok := parseTS(s)
		text := "None"
		if ok {
			text = strconv.FormatInt(got, 10)
			parsed++
		}
		if w := pyjson.Str(want[i]); w != text {
			t.Errorf("_parse_ts(%s): Python %s, Go %s", pyjson.StrRepr(s), w, text)
		}
	}
	if parsed < 1000 {
		t.Errorf("only %d of %d strings parse: the corpus misses the accepting branches", parsed, len(corpus))
	}
}
