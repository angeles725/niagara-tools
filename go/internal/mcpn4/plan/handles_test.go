package plan_test

import (
	"errors"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// removing sends op instead of the set's own (n4_set_slot's plan and read-back otherwise).
func removing(r *rig, op *pyjson.Object) {
	impl := writeops.SetSlot
	setPlan := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := setPlan(c, args)
		if err == nil {
			p.Ops = []any{op}
		}
		return p, err
	}
	panicked(r, impl)
}

// _send: a removal sent and answered makes every cached handle stale (invalidate_handles());
// any other op, or a removal the station call failed on, leaves the cache alone.
func TestARemovalClearsEveryCachedHandle(t *testing.T) {
	for _, c := range []struct {
		op      *pyjson.Object
		sendErr error
		cleared int
	}{
		{node("nm", "v", "h", "3", "n", "Calc"), nil, 1},
		{node("nm", "v", "h", "3", "n", "Calc"), errors.New("boom"), 0},
		{node("nm", "s", "h", "10", "n", "out", "b", node("nm", "p", "s", []any{})), nil, 0},
		{node("nm", "a", "h", "3", "n", "X", "b", node("nm", "p", "s", []any{})), nil, 0},
	} {
		r := newRig(t)
		removing(r, c.op)
		tok := r.dry(t, "n4_set_slot", setArgs())
		r.st.sendErr = c.sendErr
		r.execute(t, "n4_set_slot", setArgs(), tok)
		if r.st.cleared != c.cleared {
			t.Errorf("%s (send error %v): ClearHandles called %d times, want %d", pyjson.Repr(c.op), c.sendErr,
				r.st.cleared, c.cleared)
		}
	}
}
