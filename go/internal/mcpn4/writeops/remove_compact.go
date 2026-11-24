package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// removeReadback is _remove_readback: the child must be gone from its parent.
func removeReadback(c plan.Client, args *pyjson.Object, _ *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	nm := str(args, "name")
	_, still := nodes.Get(nm)
	v := "verified"
	if still {
		v = "mismatch"
	}
	return obj("removed", nm), replies[0], obj("gone", !still), v, nil
}

// firstObj is the first op of ops when it is an object.
func firstObj(ops []any) (*pyjson.Object, bool) {
	if len(ops) == 0 {
		return nil, false
	}
	o, ok := ops[0].(*pyjson.Object)
	return o, ok && o != nil
}

// inverseSummary is plan.inverse_summary of a compact dry run: what the snapshot holds, counted.
func inverseSummary(inverse []any, first *pyjson.Object) (*pyjson.Object, error) {
	_, nested, err := flatten(0, get(first, "b"))
	if err != nil {
		return nil, err
	}
	_, relinks, unlinked, err := inverseParts(inverse)
	if err != nil {
		return nil, err
	}
	outgoing, err := outgoingParts(inverse)
	if err != nil {
		return nil, err
	}
	return obj("components", 1+len(nested), "links_inside", len(relinks), "unlinked_inputs", len(unlinked),
		"outgoing_links", len(outgoing), "inverse_ops", len(inverse)), nil
}
