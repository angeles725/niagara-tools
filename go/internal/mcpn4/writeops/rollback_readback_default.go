package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// restoredOrDefault is restored on nodes that hold the declared default of a slot the load left
// out (plan.Resolve): a frozen property set back to its default is omitted again, so it reads as
// restored when that default is the recorded value (lab retro 2026-11-15 row 2). A type
// contract that cannot be read is a read-back error (readback_error), not a station that differs.
func restoredOrDefault(c plan.Client, nodes, op *pyjson.Object) (bool, error) {
	if slot, ok := get(op, "n").(string); ok {
		filled, err := plan.Resolve(c, nodes, slot)
		if err != nil {
			return false, err
		}
		nodes = filled
	}
	return restored(nodes, op)
}
