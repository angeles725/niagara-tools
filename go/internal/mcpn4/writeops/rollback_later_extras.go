package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
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

// vanishedWith marks in undone the batches a completed rollback of a single-component create took
// with the component: those journaled between the create and its rollback that wrote only under it
// (a chain=true rollback removes them with it and gives them no rollback of their own), on the same
// station. Work journaled after the rollback is not marked, and neither is a failed rollback's.
func vanishedWith(views []*pyjson.Object, undone map[string]bool) {
	index := map[string]int{}
	for i, v := range views {
		if id, _ := get(v, "batch_id").(string); id != "" {
			if _, seen := index[id]; !seen {
				index[id] = i
			}
		}
	}
	for r, v := range views {
		of, _ := get(v, "rollback_of").(string)
		c, ok := index[of]
		if !ok || c >= r || get(v, "state") != "completed" || get(v, "verdict") == "failed" {
			continue
		}
		created, ok := createdComponent(views[c])
		if !ok {
			continue
		}
		under, err := ord.NewScope([]string{created})
		if err != nil {
			continue
		}
		for _, b := range views[c+1 : r] {
			t, _ := get(b, "targets").(*pyjson.Object)
			if id, _ := get(b, "batch_id").(string); id != "" && t != nil && box.PyEqual(get(b, "station_name"), get(views[c], "station_name")) &&
				allUnder(under, t.Keys()) {
				undone[id] = true
			}
		}
	}
}

// createdComponent is the ORD a create batch made, from its recorded inverse (one remove op); false
// for any other batch.
func createdComponent(view *pyjson.Object) (string, bool) {
	if tool := get(view, "tool"); tool != createTool && tool != registerTool {
		return "", false
	}
	inverse, _ := get(view, "inverse").([]any)
	targets, _ := get(view, "targets").(*pyjson.Object)
	if len(inverse) != 1 || targets == nil {
		return "", false
	}
	op, _ := inverse[0].(*pyjson.Object)
	if op == nil || get(op, "nm") != "v" {
		return "", false
	}
	created, err := childOrd(ordOf(targets, get(op, "h")), get(op, "n"))
	return created, err == nil
}

// allUnder is whether every one of ords is under scope (at least one).
func allUnder(scope *ord.Scope, ords []string) bool {
	for _, o := range ords {
		if scope.Check(o) != nil {
			return false
		}
	}
	return len(ords) > 0
}
