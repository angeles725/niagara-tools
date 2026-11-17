package writeops

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func node(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// A runtime slot the snapshot classified but that left the body (a link-driven slot moves to
// prior) marks nothing: no hashed copy, so the plan carries no runtime note (row M15b).
func TestHashedAddIsNilWhenNoRuntimeSlotStaysInTheBody(t *testing.T) {
	gone := node(t, `{"n": "out", "v": "1"}`)
	s := &snapshot{runtime: map[*pyjson.Object]bool{gone: true}}
	add := node(t, `{"nm": "a", "b": {"n": "P", "s": [{"n": "timeDelay", "v": "5000"}]}}`)
	if got := s.hashedAdd(add); got != nil {
		t.Errorf("hashedAdd = %v, want nil", got)
	}
	kid := get(add, "b").(*pyjson.Object)
	tick := get(kid, "s").([]any)[0].(*pyjson.Object)
	s.runtime[tick] = true
	if got := s.hashedAdd(add); got == nil || !marked(get(got, "b").(*pyjson.Object)) {
		t.Errorf("hashedAdd = %v, want the marked copy", got)
	}
}

func TestFirstObjRefusesAnEmptyOrForeignInverse(t *testing.T) {
	if _, ok := firstObj(nil); ok {
		t.Error("empty inverse accepted")
	}
	if _, ok := firstObj([]any{"x"}); ok {
		t.Error("a non-object op accepted")
	}
}
