package jsonrpc

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
)

func nestedPing(depth int) string {
	return `{"id":` + strings.Repeat("[", depth-1) + "1" + strings.Repeat("]", depth-1) + `,"method":"ping"}`
}

func TestNestingUpToTheLimitIsAnswered(t *testing.T) {
	t.Parallel()
	got := handle(t, New(registry.Default()), nestedPing(MaxNesting))
	if !strings.HasSuffix(got, `, "result": {}}`) {
		t.Errorf("depth %d: %.80s...", MaxNesting, got)
	}
}

func TestDeeperNestingIsAParseErrorAndTheServerKeepsServing(t *testing.T) {
	t.Parallel()
	s := New(registry.Default())
	const parseError = `{"jsonrpc": "2.0", "id": null, "error": {"code": -32700, "message": "parse error"}}`
	for _, depth := range []int{MaxNesting + 1, 1 << 20} {
		for _, line := range []string{nestedPing(depth), strings.Repeat(`{"a":`, depth) + "1" + strings.Repeat("}", depth)} {
			if got := handle(t, s, line); got != parseError {
				t.Errorf("depth %d: %.80s", depth, got)
			}
		}
	}
	if got := handle(t, s, `{"id":1,"method":"ping"}`); got != `{"jsonrpc": "2.0", "id": 1, "result": {}}` {
		t.Errorf("after: %s", got)
	}
}

func TestNestsDeeperSkipsStringContents(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]bool{
		`[[1]]`:              false,
		`[[[1]]]`:            true,
		`["[[[[", "\"[[[["]`: false,
		`[{"k": [1]}]`:       true,
		`[1], [2], [3]`:      false,
	} {
		if got := NestsDeeper([]byte(text), 2); got != want {
			t.Errorf("NestsDeeper(%s, 2) = %v, want %v", text, got, want)
		}
	}
}
