package pyjson

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pytarget"
)

// TestDifferentialPython is the equivalence proof against CPython's json module: every boundary
// case plus a fuzz corpus from a fixed seed goes through json.loads, then json.dumps (default)
// and safety.canonical, then float() for a top-level number; random float bit patterns go
// through repr(). python3 decodes the input bytes with surrogateescape, as its stdin does in
// UTF-8 mode (the default since 3.15). The Go result must equal Python's byte for byte.
// It skips, naming the Python found, unless that is the target CPython (pytarget.Minor).

const pyDriver = `
import json, struct, sys
for line in open(sys.argv[1]):
    kind, data = line.rstrip("\n").split(" ", 1)
    raw = bytes.fromhex(data)
    if kind == "F":
        print(repr(struct.unpack("<d", raw)[0]))
        continue
    try:
        v = json.loads(raw.decode("utf-8", "surrogateescape"))
    except ValueError:
        print("E")
        continue
    conv = "-"
    if type(v) in (int, float):
        try:
            conv = repr(float(v))
        except OverflowError as e:
            conv = "OverflowError: %s" % e
    print("%s\t%s\t%s" % (json.dumps(v), json.dumps(v, sort_keys=True, separators=(",", ":")), conv))
`

const fuzzSeed, fuzzDocs, fuzzFloats = 20261022, 4000, 4000

func TestDifferentialPython(t *testing.T) {
	py := pytarget.Python(t)
	rng := rand.New(rand.NewPCG(fuzzSeed, fuzzSeed))
	var docs []string
	for _, c := range decodeCases {
		docs = append(docs, c.in)
	}
	g := gen{rng}
	for range fuzzDocs {
		docs = append(docs, g.doc())
	}
	floats := make([]uint64, fuzzFloats)
	for i := range floats {
		floats[i] = g.floatBits()
	}

	var in strings.Builder
	for _, d := range docs {
		in.WriteString("J " + hex.EncodeToString([]byte(d)) + "\n")
	}
	var b [8]byte
	for _, f := range floats {
		binary.LittleEndian.PutUint64(b[:], f)
		in.WriteString("F " + hex.EncodeToString(b[:]) + "\n")
	}
	path := filepath.Join(t.TempDir(), "corpus.txt")
	if err := os.WriteFile(path, []byte(in.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(py, "-I", "-c", pyDriver, path)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("python3 driver: %v\n%s", err, stderr.String())
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<20)
	var want []string
	for sc.Scan() {
		want = append(want, sc.Text())
	}
	if len(want) != len(docs)+len(floats) {
		t.Fatalf("python3 printed %d lines, want %d", len(want), len(docs)+len(floats))
	}
	fails := 0
	check := func(label, got, want string) {
		if got != want && fails < 10 {
			fails++
			t.Errorf("%s:\n go  %q\n py  %q", label, got, want)
		}
	}
	for i, d := range docs {
		check("doc "+hex.EncodeToString([]byte(d)), goLine(d), want[i])
	}
	for i, f := range floats {
		check("float bits "+hex.EncodeToString(binary.LittleEndian.AppendUint64(nil, f)),
			FloatRepr(math.Float64frombits(f)), want[len(docs)+i])
	}
}

// goLine is the driver's output line computed by this package.
func goLine(doc string) string {
	v, err := Unmarshal([]byte(doc))
	if err != nil {
		return "E"
	}
	dumps, err1 := Marshal(v, Options{})
	canon, err2 := Marshal(v, Canonical)
	if err1 != nil || err2 != nil {
		return "marshal error"
	}
	conv := "-"
	if n, ok := v.(Number); ok {
		if f, err := n.Float64(); err != nil {
			conv = "OverflowError: " + err.Error()
		} else {
			conv = FloatRepr(f)
		}
	}
	return string(dumps) + "\t" + string(canon) + "\t" + conv
}

// gen builds JSON-ish documents: mostly valid, with edge numbers, every string escape, lone
// surrogates, raw non-ASCII and invalid bytes, duplicate keys, and a share of byte mutations
// that make them invalid. Nesting stays shallow: Python's RecursionError is out of scope.
type gen struct{ r *rand.Rand }

var edgeNumbers = []string{
	"0", "-0", "-0.0", "0.0", "100", "100.0", "1e16", "1e15", "1E+16", "1e-5", "0.0001",
	"0.00001", "1e400", "-1e400", "1e-400", "1e308", "1.7976931348623157e308",
	"1.7976931348623159e308", "5e-324", "2.4703282292062327e-324", "9007199254740993",
	"9007199254740993.0", "123456789012345678", "0.1", "1.5e-07", "12345678901234567890",
	"1" + strings.Repeat("0", 308), "1" + strings.Repeat("0", 400), "-1" + strings.Repeat("0", 400),
	"NaN", "Infinity", "-Infinity", "1.0E-0", "-12.5e+003",
}

var pieces = []string{
	`\"`, `\\`, `\/`, `\b`, `\f`, `\n`, `\r`, `\t`, `\u0000`, `\u001F`, `\u007f`, `\u00e9`,
	`\uD83D\uDE00`, `\ud800`, `\udfff`, `\ud83d\u0041`, `\udc00\ud800`, "\u00e9", "\u2028",
	"\U0001f600", "\uffff", "<", ">", "&", "'", "a", "Z", " ", "\x7f", "\xff", "\xed\xa0\x80",
	"\x01", `\x`, `\u12`,
}

// mutations are the bytes a mutation inserts.
const mutations = ",]}:\"e.-0 x"

func (g gen) doc() string {
	s := g.ws() + g.value(0) + g.ws()
	if g.r.IntN(5) == 0 { // mutate: drop, duplicate or insert one byte
		i := g.r.IntN(len(s) + 1)
		switch g.r.IntN(3) {
		case 0:
			if i < len(s) {
				s = s[:i] + s[i+1:]
			}
		case 1:
			if i < len(s) {
				s = s[:i] + s[i:i+1] + s[i:]
			}
		default:
			s = s[:i] + string(mutations[g.r.IntN(len(mutations))]) + s[i:]
		}
	}
	return s
}

func (g gen) ws() string {
	return strings.Repeat(string(" \t\n\r"[g.r.IntN(4)]), g.r.IntN(3))
}

func (g gen) value(depth int) string {
	n := 7
	if depth >= 4 {
		n = 5
	}
	switch g.r.IntN(n) {
	case 0:
		return []string{"null", "true", "false"}[g.r.IntN(3)]
	case 1:
		return g.number()
	case 2, 3:
		return g.str()
	case 4:
		return g.number()
	case 5:
		var items []string
		for range g.r.IntN(4) {
			items = append(items, g.ws()+g.value(depth+1)+g.ws())
		}
		return "[" + strings.Join(items, ",") + "]"
	default:
		var items []string
		keys := []string{`"a"`, `"b"`, `"B"`, `"\u00e9"`, `"z"`, "\"\U0001f600\"", `"\ud800"`}
		for range g.r.IntN(5) {
			k := keys[g.r.IntN(len(keys))]
			if g.r.IntN(2) == 0 {
				k = g.str()
			}
			items = append(items, g.ws()+k+g.ws()+":"+g.ws()+g.value(depth+1))
		}
		return "{" + strings.Join(items, ",") + "}"
	}
}

func (g gen) number() string {
	if g.r.IntN(2) == 0 {
		return edgeNumbers[g.r.IntN(len(edgeNumbers))]
	}
	digits := func(n int) string {
		var b strings.Builder
		for range n {
			b.WriteByte(byte('0' + g.r.IntN(10)))
		}
		return b.String()
	}
	s := strings.TrimLeft(digits(1+g.r.IntN(20)), "0")
	if s == "" {
		s = "0"
	}
	if g.r.IntN(3) == 0 {
		s = "-" + s
	}
	if g.r.IntN(2) == 0 {
		s += "." + digits(1+g.r.IntN(18))
	}
	if g.r.IntN(3) == 0 {
		s += []string{"e", "E", "e+", "e-", "E-"}[g.r.IntN(5)] + digits(1+g.r.IntN(3))
	}
	return s
}

func (g gen) str() string {
	var b strings.Builder
	b.WriteByte('"')
	for range g.r.IntN(6) {
		p := pieces[g.r.IntN(len(pieces))]
		if g.r.IntN(8) != 0 && (p == "\x01" || p == `\x` || p == `\u12`) {
			p = "q" // keep most strings valid
		}
		b.WriteString(p)
	}
	b.WriteByte('"')
	return b.String()
}

// floatBits mixes uniform bit patterns (all exponents, subnormals, nan payloads) with
// decimal-looking values whose repr sits near the fixed/exponent switch.
func (g gen) floatBits() uint64 {
	if g.r.IntN(2) == 0 {
		return g.r.Uint64()
	}
	f := float64(g.r.Int64N(1_000_000_000)) * math.Pow(10, float64(g.r.IntN(44)-22))
	if g.r.IntN(2) == 0 {
		f = -f
	}
	return math.Float64bits(f)
}
