package jsonrpc

import (
	"bufio"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

// TestDifferentialPython is the protocol-level equivalence proof: every line below goes through
// the Python Server(...).handle_line (the in-process read-only server, decoding the bytes as its
// stdin does) and through HandleLine; the reply bytes must be equal. Tool calls stop at the
// layers ported in S2 (validation, the session gate), so no station is needed.
// Python's replies are recorded in testdata/handle-line.golden.jsonl.

const handleLineDriver = `
import sys
from mcp_n4 import server
srv = server.Server()
for line in sys.stdin:
    raw = bytes.fromhex(line.strip())
    out = srv.handle_line(raw.decode("utf-8", "surrogateescape"))
    print("-" if out is None else out)
`

var protocolLines = []string{
	``, `   `, "\t\v\f\x1c\u00a0\u3000", `{not json`, `[1, 2]`, `5`, `"x"`, `null`, `{}`,
	`{"id": 1}`, `{"id": 1, "method": null}`, `{"id": 1, "method": 5}`, `{"id": 1, "method": [1, "a"]}`,
	`{"id": 1, "method": {"a": 1.0}}`, `{"id": 1, "method": ""}`, `{"id": 1, "method": "nope"}`,
	`{"method": "nope"}`, `{"jsonrpc": "2.0", "method": "notifications/initialized"}`,
	`{"jsonrpc":"2.0","id":null,"method":"ping"}`, `{"id": 1.5e300, "method": "ping"}`,
	`{"id": NaN, "method": "ping"}`, `{"id": -Infinity, "method": "ping"}`, `{"id": "\u00e9\ud83d\ude00", "method": "ping"}`,
	`{"id": "\ud800", "method": "ping"}`, `{"id": [{"a": [true, false, null]}], "method": "ping"}`,
	`{"id": 123456789012345678901234567890, "method": "ping"}`, `{"id": 1, "id": 2, "method": "ping"}`,
	"\u2028{\"id\": 7, \"method\": \"ping\"}\u0085", "\ufeff{\"id\": 7, \"method\": \"ping\"}",
	"{\"id\": \"\xff\", \"method\": \"ping\"}", "\xff", `{"id": 1, "method": "ping", "params": 5}`,
	`{"id": 1, "method": "initialize"}`, `{"id": 1, "method": "initialize", "params": {"protocolVersion": "2025-03-26"}}`,
	`{"id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05"}}`,
	`{"id": 1, "method": "initialize", "params": {"protocolVersion": "2025-11-25"}}`,
	`{"id": 1, "method": "initialize", "params": {"protocolVersion": 5}}`, `{"id": 1, "method": "initialize", "params": []}`,
	`{"id": 2, "method": "tools/list"}`, `{"id": 2, "method": "tools/list", "params": {"cursor": "x"}}`,
	`{"id": 3, "method": "tools/call"}`, `{"id": 3, "method": "tools/call", "params": {}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": 5}}`, `{"id": 3, "method": "tools/call", "params": {"name": "ghost"}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_set_slot", "arguments": {}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": []}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": null}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate"}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": 4}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": 0}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": "1"}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": true}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": 2.0}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": 99999999999999999999999}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"depth": 1, "types": "x"}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_navigate", "arguments": {"bogus": {"deep": [1]}, "depth": 1}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_connect", "arguments": {}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_connect", "arguments": {"station": 5}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_connect", "arguments": {"station": "x", "expected_station": 7}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_bql_query", "arguments": {"query": "select"}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_list_batches", "arguments": {"limit": 0}}}`,
	`{"id": 3, "method": "tools/call", "params": {"name": "n4_list_batches", "arguments": {"limit": 1e9}}}`,
}

func TestDifferentialPython(t *testing.T) {
	g := golden.Open(t, "handle-line")
	lines := append(append([]string{}, protocolLines...), nestedPing(MaxNesting), nestedPing(MaxNesting+1),
		strings.Repeat(`{"a":`, 100000)+"1"+strings.Repeat("}", 100000)) // past CPython's C stack
	var in strings.Builder
	for _, l := range lines {
		in.WriteString(hex.EncodeToString([]byte(l)) + "\n")
	}
	out := g.Run(handleLineDriver, []byte(in.String()))
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Buffer(nil, 1<<24)
	s := New(registry.Default())
	i := 0
	for ; sc.Scan(); i++ {
		if i >= len(lines) {
			t.Fatalf("python3 printed more lines than the corpus holds")
		}
		want := sc.Text()
		got := "-"
		if reply, ok := s.HandleLine([]byte(lines[i])); ok {
			got = string(reply)
		}
		if got != want {
			t.Errorf("line %q:\n go     %s\n python %s", lines[i], got, want)
		}
	}
	if i != len(lines) {
		t.Fatalf("python3 answered %d of %d lines", i, len(lines))
	}
}
