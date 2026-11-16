package writeops

import (
	"errors"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The rollback of a create removes the created component and everything under it. It is refused
// when that would silently destroy later work (bench retro 2026-11-15, delta 1): a later batch
// journaled under the component, or a child or link on it that its create did not make (a
// Workbench edit no journal has).

// createTool is the tool whose rollback is a remove of what it created.
const createTool = "n4_create_component"

// viewsJournal is the journal's merged batch views, in journal order (journal.Journal.Views).
type viewsJournal interface {
	Views() ([]*pyjson.Object, error)
}

// laterWork refuses the rollback of create batch bid (its view, its single remove op and its
// recorded targets) when later work sits under the component it created; any other batch passes.
func laterWork(c plan.Client, w *plan.State, view *pyjson.Object, ops []any, targets *pyjson.Object, bid string) error {
	if get(view, "tool") != createTool || len(ops) != 1 {
		return nil
	}
	remove := ops[0].(*pyjson.Object)
	created, err := childOrd(ordOf(targets, get(remove, "h")), get(remove, "n"))
	if err != nil {
		return err
	}
	batches, err := laterBatches(w, view, bid, created)
	if err != nil {
		return err
	}
	extras, err := extraChildren(c, view, ordOf(targets, get(remove, "h")), get(remove, "n").(string))
	if err != nil {
		return err
	}
	if len(batches) == 0 && len(extras) == 0 {
		return nil
	}
	return plan.ToolError(laterWorkRefusal(bid, created, batches, extras))
}

// laterBatches is "<batch id> (<tool>)" of every batch journaled after bid on the same station,
// newest first, whose recorded targets are created or under it; a rollback batch, and a batch
// undone by a completed rollback that did not fail, are no later work.
func laterBatches(w *plan.State, view *pyjson.Object, bid, created string) ([]string, error) {
	j, ok := w.Journal.(viewsJournal)
	if !ok {
		return nil, &plan.PyError{Type: "AttributeError", Msg: "the journal cannot be read"}
	}
	views, err := j.Views()
	if err != nil {
		return nil, err
	}
	under, err := ord.NewScope([]string{created}) // the write scope's slot-boundary rule
	if err != nil {
		return nil, err
	}
	undone, after := map[string]bool{}, -1
	for i, v := range views {
		if of, ok := get(v, "rollback_of").(string); ok && get(v, "state") == "completed" && get(v, "verdict") != "failed" {
			undone[of] = true
		}
		if id, _ := get(v, "batch_id").(string); id == bid && after < 0 {
			after = i
		}
	}
	var out []string
	for i := len(views) - 1; i > after && after >= 0; i-- {
		v := views[i]
		id, _ := get(v, "batch_id").(string)
		if pyjson.Truthy(get(v, "rollback_of")) || undone[id] || !box.PyEqual(get(v, "station_name"), get(view, "station_name")) {
			continue
		}
		if t, ok := get(v, "targets").(*pyjson.Object); ok && anyUnder(under, t.Keys()) {
			out = append(out, id+" ("+pyjson.Str(get(v, "tool"))+")")
		}
	}
	return out, nil
}

func anyUnder(scope *ord.Scope, ords []string) bool {
	for _, o := range ords {
		if scope.Check(o) == nil {
			return true
		}
	}
	return false
}

// extraChildren is the names of the links and component children the component name under
// parent now holds that its create did not make: not a slot of the create's body and not a frozen
// slot its type declares. The parent is loaded two deep, as the create's read-back loads it: a
// component no longer there holds nothing, and the remove reports it as before.
func extraChildren(c plan.Client, view *pyjson.Object, parent, name string) ([]string, error) {
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
	var out []string
	kids := box.Children(nodes, name)
	for _, n := range kids.Keys() {
		kid := box.Node(kids, n)
		t := get(kid, "t")
		if isLink(t) {
			out = append(out, n)
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
			out = append(out, n)
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
