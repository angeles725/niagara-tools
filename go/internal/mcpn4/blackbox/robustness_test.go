package blackbox

// Byte-level replies (port of test_server.TestRobustness; the id-null, no-id and nesting tests
// are pinned in-process by jsonrpc.TestIdNullIsARequestAndGetsAReply,
// jsonrpc.TestMessageWithoutIdIsStillANotification, jsonrpc.TestNestingUpToTheLimitIsAnswered and
// jsonrpc.TestDeeperNestingIsAParseErrorAndTheServerKeepsServing).

import (
	"encoding/json"
	"os"
	"testing"
)

// TestRepliesAreTheBytesThePythonServerWrites compares tools/list (every schema), errors and odd
// ids, byte for byte, with what the Python server wrote; testdata/protocol.json is the frozen
// recording (mcp-n4-kit/tests/fixtures/oracles/protocol.json, recorded before S17 removed that
// server; it cannot be re-recorded).
func TestRepliesAreTheBytesThePythonServerWrites(t *testing.T) {
	data, err := os.ReadFile("testdata/protocol.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Calls map[string][][2]string `json:"calls"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	calls := doc.Calls["TestRobustness.test_replies_are_the_bytes_the_python_server_writes"]
	if len(calls) != 9 {
		t.Fatalf("%d recorded calls, want 9", len(calls))
	}
	s := Start(t, Options{})
	for _, c := range calls {
		if got := s.Line(c[0]); got != c[1] {
			t.Errorf("%q:\n got %.200s\nwant %.200s", c[0], got, c[1])
		}
	}
}

func TestWhitespaceAroundALineIsPythonStrStrip(t *testing.T) {
	s := Start(t, Options{})
	if got := s.Line("\u3000\x1c\x0c"); got != "" {
		t.Errorf("whitespace-only line answered %s", got)
	}
	if got := s.decode(s.Line("\u2028{\"id\":1,\"method\":\"ping\"}\u0085")); !isEmptyObject(got["result"]) {
		t.Errorf("reply %v", got)
	}
}
