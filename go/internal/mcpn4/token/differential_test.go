package token

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The Python side injects the key, clock and pending nonces into one ConfirmationTokens
// instance; the installed package is never patched.
const helperScript = `import json, sys
from mcp_n4 import safety
cases = json.load(sys.stdin)
out = []
for c in cases:
    now = c["now"]
    tok = safety.ConfirmationTokens(ttl=c["ttl"], clock=lambda: now)
    tok._key = bytes.fromhex(c["key"])
    tok._pending = dict(c["pending"])
    out.append(tok._mac(c["tool"], c["args"], c["plan"], c["expiry"], c["nonce"]))
    out.append(safety.canonical(safety.bound_args(c["args"])))
    out.append(json.dumps(safety.redact_args(c["args"])))
    try:
        tok.consume(c["tool"], c["args"], c["plan"], c["token"])
        out.append("ok")
    except safety.SafetyError as exc:
        out.append("refused: %s" % exc)
    out.append(sorted(tok._pending.items()))
    token, expiry = tok.issue(c["tool"], c["args"], c["plan"])
    fresh = token.split(".")[1]
    out.append([expiry, sorted(k for k in tok._pending if k != fresh)])
print(json.dumps(out))`

// argSets are the args the tokens are bound to, as JSON: key order, Unicode and surrogate keys,
// floats Python writes its own way, big ints, and the two unbound keys.
var argSets = []string{
	`{"ord": "station:|slot:/A", "slot": "out"}`,
	`{"slot": "out", "ord": "station:|slot:/A", "dry_run": true, "confirmation_token": "x"}`,
	`{"v": 1.0, "w": 1e16, "x": 1e-05, "y": -0.0, "z": 123456789012345678901234567890, "n": NaN, "i": -Infinity}`,
	`{"é": [1, {"b": null, "a": false}], "\ud800": "\udfff", "\ud83d\ude00": " ", "A": "<&>"}`,
	`{"password": "p", "nested": {"Token": 1, "TOKEN": 2, "credentİal": 3}, "list": [{"api_secret": [1]}]}`,
	`{}`,
}

// tokenCase is one consume probe: the token, the clock and the pending nonces before it.
type tokenCase struct {
	token   any
	now     float64
	pending map[string]int64
}

func TestTokensDifferentialPython(t *testing.T) {
	g := golden.Open(t, "tokens")
	key := strings.Repeat("ab", 16) + strings.Repeat("0f", 16)
	var cases []any
	var gotGo []any
	for ai, argText := range argSets {
		argsV, err := pyjson.Loads([]byte(argText))
		if err != nil {
			t.Fatal(err)
		}
		args := argsV.(*pyjson.Object)
		probe := New(300, nil)
		probe.key, _ = hex.DecodeString(key)
		mac, _ := probe.mac("n4_set_slot", args, "p1", 1300, "00112233aabbccdd")
		good := "1300.00112233aabbccdd." + mac
		for _, tc := range []tokenCase{
			{good, 1000, map[string]int64{"00112233aabbccdd": 1300}},
			{good, 1299.5, map[string]int64{"00112233aabbccdd": 1300, "old": 900}},
			{good, 1300, map[string]int64{"00112233aabbccdd": 1300}},
			{good, 1000, map[string]int64{}},
			{"0001300.00112233aabbccdd." + mac, 1000, map[string]int64{"00112233aabbccdd": 1300}},
			{"1301.00112233aabbccdd." + mac, 1000, map[string]int64{"00112233aabbccdd": 1301}},
			{strings.ToUpper(good), 1000, map[string]int64{"00112233aabbccdd": 1300}},
			{good + ".x", 1000, nil}, {"١.a.b", 1000, nil}, {nil, 1000, nil}, {7, 1000, nil}, {"", 1000, nil},
			{"1.\xed\xa0\x80." + mac, 1000, nil}, {"1.." + mac, 1000, map[string]int64{"": 2}},
		} {
			for _, tool := range []string{"n4_set_slot", "n4_create_link"} {
				c := pyjson.NewObject()
				c.Set("now", pyjson.FloatNumber(tc.now))
				c.Set("ttl", 300)
				c.Set("key", key)
				// Sorted, not map order: the cases are recorded in a golden that replays byte for byte.
				c.Set("pending", sortedPending(tc.pending))
				c.Set("tool", tool)
				c.Set("args", args)
				c.Set("plan", "p1")
				c.Set("expiry", 1300+int64(ai))
				c.Set("nonce", "né-"+tool)
				c.Set("token", tc.token)
				cases = append(cases, c)
				gotGo = append(gotGo, goSide(t, key, tc, tool, args, 1300+int64(ai), "né-"+tool)...)
			}
		}
	}
	data, err := pyjson.Marshal(cases, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	out := g.Run(helperScript, data)
	want := strings.TrimSpace(string(out))
	got, _ := pyjson.Marshal(gotGo, pyjson.Options{})
	if string(got) != want {
		t.Fatalf("Go and Python differ\ngo:     %.3000s\npython: %.3000s", got, want)
	}
	if len(gotGo) != 6*len(cases) {
		t.Fatalf("%d results for %d cases", len(gotGo), len(cases))
	}
}

// goSide is what the helper prints for one case, computed by the Go tokens.
func goSide(t *testing.T, key string, tc tokenCase, tool string, args *pyjson.Object, expiry int64, nonce string) []any {
	now := tc.now
	tok := New(300, func() float64 { return now })
	tok.key, _ = hex.DecodeString(key)
	for n, e := range tc.pending {
		tok.pending[n] = e
	}
	mac, err := tok.mac(tool, args, "p1", expiry, nonce)
	if err != nil {
		t.Fatal(err)
	}
	bound, _ := pyjson.Marshal(BoundArgs(args), pyjson.Canonical)
	red, _ := pyjson.Marshal(Redact(args), pyjson.Options{})
	res := "ok"
	if err := tok.Consume(tool, args, "p1", tc.token); err != nil {
		res = "refused: " + err.Error()
	}
	after := sortedPending(tok.pending)
	fresh, exp, err := tok.Issue(tool, args, "p1")
	if err != nil {
		t.Fatal(err)
	}
	names := []any{}
	for _, kv := range sortedPending(tok.pending) {
		if n := kv.([]any)[0]; n != strings.Split(fresh, ".")[1] {
			names = append(names, n)
		}
	}
	return []any{mac, string(bound), string(red), res, after, []any{exp, names}}
}

// sortedPending is sorted(pending.items()) as Python lists it: [nonce, expiry] pairs by nonce.
func sortedPending(p map[string]int64) []any {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	out := []any{}
	for _, k := range keys {
		out = append(out, []any{k, p[k]})
	}
	return out
}
