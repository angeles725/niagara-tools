package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// extraChild is a child of a created component its create did not make.
type extraChild struct {
	name string
	link bool
}

// extraChildren is the names of the links and component children the component name under
// parent now holds that its create did not make: not a slot of the create's body and not a frozen
// slot its type declares. The parent is loaded two deep, as the create's read-back loads it: a
// component no longer there holds nothing, and the remove reports it as before.
func extraChildren(c plan.Client, view *pyjson.Object, parent, name string) ([]extraChild, error) {
	_, nodes, err := plan.Handle(c, parent, 2)
	if err != nil {
		return nil, err
	}
	node := box.Node(nodes, name)
	if node == nil {
		return nil, nil
	}
	typ, _ := get(node, "t").(string)
	made := createdSlots(view)
	var contracts *plan.Contracts
	if tl, ok := c.(plan.TypeLoader); ok && typ != "" {
		contracts = plan.NewContracts(tl)
	}
	var out []extraChild
	kids := box.Children(nodes, name)
	for _, n := range kids.Keys() {
		kid := box.Node(kids, n)
		t := get(kid, "t")
		if isLink(t) {
			out = append(out, extraChild{n, true})
			continue
		}
		if !box.IsComponentType(t) || made[n] {
			continue
		}
		frozen, err := declaredFrozen(contracts, typ, kid)
		if err != nil {
			return nil, err
		}
		if !frozen {
			out = append(out, extraChild{n, false})
		}
	}
	return out, nil
}

// createdSlots is the slot names of the create op's body ("b": {s: [{n}]}).
func createdSlots(view *pyjson.Object) map[string]bool {
	made := map[string]bool{}
	ops, _ := get(view, "ops").([]any)
	if len(ops) == 0 {
		return made
	}
	op, _ := ops[0].(*pyjson.Object)
	body, _ := get(op, "b").(*pyjson.Object)
	slots, _ := get(body, "s").([]any)
	for _, s := range slots {
		if slot, ok := s.(*pyjson.Object); ok {
			if n, ok := get(slot, "n").(string); ok {
				made[n] = true
			}
		}
	}
	return made
}

// declaredFrozen reports whether typ's contract declares kid (a frozen slot such as a writable's
// proxyExt, created with its parent). Without a contract to read it is not proven frozen.
func declaredFrozen(contracts *plan.Contracts, typ string, kid *pyjson.Object) (bool, error) {
	if contracts == nil {
		return false, nil
	}
	slot, err := contracts.Declared(typ, kid)
	var unanswered *plan.UnansweredType
	if errors.As(err, &unanswered) {
		return false, nil
	}
	return slot != nil, err
}
