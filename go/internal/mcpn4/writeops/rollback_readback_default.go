package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// restoredOrDefault is restored on nodes that hold the declared default of a slot the load left
// out (plan.Resolve): a frozen property set back to its default is omitted again, so it reads as
// restored when that default is the recorded value (lab retro 2026-11-15 row 2). A type
// contract that cannot be read is a read-back error (readback_error), not a station that differs.
func restoredOrDefault(c plan.Client, nodes, op *pyjson.Object) (bool, error) {
	if get(op, "nm") == "r" {
		return renamedBack(nodes, op)
	}
	if get(op, "nm") == "o" {
		return reorderedBack(nodes, op), nil
	}
	if slot, ok := get(op, "n").(string); ok {
		filled, err := plan.Resolve(c, nodes, slot)
		if err != nil {
			return false, err
		}
		nodes = filled
	}
	return restored(nodes, op)
}

// loadNameOnlyResolved loads where for the rollback guard. A slot the station lists by name only
// (box.ByNameOnly: a linked frozen property set back to its default, #890) takes its declared
// default, so the guard compares that default instead of reading null as a later change. An
// omitted slot is left as before.
func loadNameOnlyResolved(c plan.Client, where string, depth int, slot string) (*pyjson.Object, error) {
	nodes, err := c.LoadTree(where, depth, box.LoadOptions{})
	if err != nil || !box.ByNameOnly(box.Node(nodes, slot)) {
		return nodes, err
	}
	return plan.Resolve(c, nodes, slot)
}
