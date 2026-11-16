package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// carriedOutputs is body reduced to the runtimeOutputSlots Status children it carries (the
// configured outputs a snapshot kept, such as a constant block's out); nil when it carries none.
func carriedOutputs(body *pyjson.Object) *pyjson.Object {
	var kept []any
	for _, k := range objects(get(body, "s")) {
		if runtimeOutputSlots[name(k)] && statusType(get(k, "t")) {
			kept = append(kept, k)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	return obj("t", get(body, "t"), "s", kept)
}

// outputsHeld says whether the re-created component leaf under parentOrd holds every output
// body carries, compared as differs compares a captured frozen child; true, without a load,
// when body carries none.
func outputsHeld(c plan.Client, parentOrd string, leaf any, body *pyjson.Object) (bool, error) {
	carried := carriedOutputs(body)
	if carried == nil {
		return true, nil
	}
	path, ok := leaf.(string)
	if !ok {
		return false, nil
	}
	nodes, err := c.LoadTree(parentOrd, FrozenCheckDepth, box.LoadOptions{})
	if err != nil {
		return false, err
	}
	gaps, err := differs(nodes, path, carried)
	return err == nil && len(gaps) == 0, err
}

// andOutputsHeld is good && outputsHeld(...): the outputs are read only for a component found good.
func andOutputsHeld(c plan.Client, good bool, parentOrd string, leaf any, body *pyjson.Object) (bool, error) {
	if !good {
		return false, nil
	}
	return outputsHeld(c, parentOrd, leaf, body)
}
