package writeops

// #890: a batch that set a linked frozen property back to its declared default (a
// BooleanWritable's in16 fed by a schedule, false {null}) finds that slot listed by name only.
// The rollback guard reads it as that default, not as null, so rolling the batch back is not
// refused as "changed since".

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// typedRoot is scanRoot that also reads type contracts.
type typedRoot struct {
	scanRoot
	types *pyjson.Object
}

func (r typedRoot) LoadTypes([]string) (any, error) { return r.types, nil }

func TestTheRollbackGuardReadsASlotListedByNameOnlyAsItsDeclaredDefault(t *testing.T) {
	kid := func(n, typ, v string) any { return obj("nm", "p", "n", n, "t", typ, "v", v) }
	status := func(v, st string) *pyjson.Object {
		return obj("nm", "p", "t", "baja:StatusBoolean", "s",
			[]any{kid("value", "baja:Boolean", v), kid("status", "baja:Status", st)})
	}
	types := obj("control:BooleanWritable", obj("c", []any{obj("st", "p", "n", "in16", "v", status("false", "40"))}))
	nodes := obj("", obj("t", "control:BooleanWritable", "h", "5"), "in16", obj("nm", "p", "n", "in16"))
	root := typedRoot{scanRoot{nodes}, types}
	run := func(v, st string) error {
		op := obj("nm", "s", "h", "5", "n", "in16", "b", status(v, st))
		return unchangedSince(root, obj("batch_id", "b1", "ops", []any{op}), []any{op}, obj(guardOrd, "5"))
	}
	if err := run("false", "40"); err != nil {
		t.Fatalf("the batch wrote the default the station lists by name only: %v", err)
	}
	if err := run("true", "0"); err == nil {
		t.Fatal("a batch that wrote true {ok} passed against the default false {null}")
	}
}
