package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// RenameComponent is n4_rename_component: up to renameMax dynamic slots renamed in one batch, each
// sent as the station's RenameOp (javax.baja.sync.RenameOp, id 'r': the parent's handle, the old
// and the new name). Links stored by handle survive a rename; the inverse renames back in reverse
// order; the read-back loads each parent and checks the new name is there and the old one gone.
var RenameComponent = plan.Impl{ScopeOrds: renameScope, Plan: renamePlan, Readback: renameReadback}

// renameMax caps one batch, like the other batch tools.
const renameMax = 50

// msgRenameLinks is the plan note on what a rename keeps and what it can break.
const msgRenameLinks = "Only dynamic slots can be renamed (the station refuses a frozen one). Links and " +
	"history extensions stay with the component; a history id built from %parent.name% or %name% " +
	"takes the new name the next time it is resolved, and any ORD that names the old path breaks."

type renameItem struct{ parentOrd, oldName, newName string }

type renameData struct {
	items  []renameItem
	frozen []frozenName
}

// renameItems reads the items argument; it does not touch the station.
func renameItems(args *pyjson.Object) ([]renameItem, error) {
	v, _ := args.Get("items")
	list, _ := v.([]any)
	if len(list) < 1 || len(list) > renameMax {
		return nil, plan.ToolError(fmt.Sprintf("items needs 1 to %d renames, got %d", renameMax, len(list)))
	}
	out := make([]renameItem, 0, len(list))
	for _, e := range list {
		o, ok := e.(*pyjson.Object)
		if !ok {
			return nil, plan.ToolError("each item needs parent_ord, name and new_name")
		}
		oldName, err := plan.ExistingName("name", get(o, "name"))
		if err != nil {
			return nil, err
		}
		newName, err := plan.Name("new_name", get(o, "new_name"))
		if err != nil {
			return nil, err
		}
		if oldName == newName {
			return nil, plan.ToolError("component already named " + pyjson.StrRepr(newName))
		}
		out = append(out, renameItem{str(o, "parent_ord"), oldName, newName})
	}
	return out, nil
}

// renameScope is every parent the items name, once each, in order.
func renameScope(args *pyjson.Object) []any {
	v, _ := args.Get("items")
	list, _ := v.([]any)
	out, seen := []any{}, map[string]bool{}
	for _, e := range list {
		if o, ok := e.(*pyjson.Object); ok {
			if p := str(o, "parent_ord"); !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

func renamePlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	items, err := renameItems(args)
	if err != nil {
		return nil, err
	}
	handles, names, requested := map[string]any{}, map[string]map[string]bool{}, map[string]bool{}
	targets := pyjson.NewObject()
	var ops, inverse, notes []any
	for _, it := range items {
		if _, loaded := handles[it.parentOrd]; !loaded {
			h, nodes, err := plan.Handle(c, it.parentOrd, 1)
			if err != nil {
				return nil, err
			}
			handles[it.parentOrd], names[it.parentOrd] = h, childNames(nodes)
			targets.Set(it.parentOrd, h)
		}
		h, held := handles[it.parentOrd], names[it.parentOrd]
		claimed := it.parentOrd + "|" + it.newName
		if err := renameCheck(it, held, requested[claimed]); err != nil {
			return nil, err
		}
		requested[claimed] = true
		held[it.oldName], held[it.newName] = false, true // later items see this rename
		ops = append(ops, renameOp(h, it.oldName, it.newName))
		inverse = append([]any{renameOp(h, it.newName, it.oldName)}, inverse...)
		notes = append(notes, it.parentOrd+": "+it.oldName+" -> "+it.newName)
	}
	notes = append(notes, msgRenameLinks)
	frozen, historyNotes, err := renameHistory(c, items, pyjson.Truthy(get(args, "freeze_history_names")))
	if err != nil {
		return nil, err
	}
	var freeze, unfreeze []any // the names are frozen before the renames and thawed after their inverse
	for _, f := range frozen {
		freeze = append(freeze, setOp(f.h, "historyName", obj("nm", "p", "t", "baja:Format", "v", f.text)))
		unfreeze = append(unfreeze, setOp(f.h, "historyName", obj("nm", "p", "t", "baja:Format", "v", f.pattern)))
		targets.Set("station:|slot:"+f.path, f.h)
	}
	return &plan.Planned{Ops: append(freeze, ops...), Inverse: append(inverse, unfreeze...),
		Notes: append(notes, historyNotes...), Targets: targets, Data: renameData{items, frozen}}, nil
}

// renameCheck refuses a rename whose source is missing (or already renamed by an earlier item)
// or whose target is taken (by the station, or by an earlier item: twice).
func renameCheck(it renameItem, held map[string]bool, twice bool) error {
	if !held[it.oldName] {
		if _, renamed := held[it.oldName]; renamed {
			return plan.ToolError(pyjson.StrRepr(it.oldName) + " is renamed twice under " + it.parentOrd)
		}
		return plan.ToolError("component " + pyjson.StrRepr(it.oldName) + " not found under " + it.parentOrd)
	}
	if twice {
		return plan.ToolError(pyjson.StrRepr(it.newName) + " is requested twice under " + it.parentOrd)
	}
	if held[it.newName] {
		return plan.ToolError(pyjson.StrRepr(it.newName) + " already exists under " + it.parentOrd)
	}
	return nil
}

// renameOp is the BOX body of a RenameOp.
func renameOp(h any, oldName, newName string) *pyjson.Object {
	return obj("nm", "r", "h", h, "o", oldName, "n", newName)
}

// renameReadback loads each parent again: verified only when every new name is present and every
// old name is gone; observed lists, per item, the names the parent holds.
func renameReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	items, frozen := p.Data.(renameData).items, p.Data.(renameData).frozen
	verdict, requested, observed := "verified", []any{}, []any{}
	for _, it := range items {
		_, nodes, err := plan.Handle(c, it.parentOrd, 1)
		if err != nil {
			return fail(err)
		}
		held := childNames(nodes)
		requested = append(requested, obj("parent_ord", it.parentOrd, "name", it.newName))
		observed = append(observed, obj("parent_ord", it.parentOrd, "new_name_present", held[it.newName],
			"old_name_present", held[it.oldName], "old_name", it.oldName))
		if !held[it.newName] || held[it.oldName] {
			verdict = "mismatch"
		}
	}
	for _, f := range frozen { // a frozen name still holds its text under the renamed path
		_, nodes, err := plan.Handle(c, "station:|slot:"+f.newPath, 1)
		if err != nil {
			return fail(err)
		}
		got, err := plan.Observe(nodes, "historyName", "baja:Format")
		if err != nil {
			return fail(err)
		}
		requested = append(requested, obj("history_ext", f.newPath, "historyName", f.text))
		observed = append(observed, obj("history_ext", f.newPath, "historyName", got))
		if got != f.text {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

// renamedBack is the rollback read-back of a RenameOp: the name it restores ("n") is on the
// parent again and the name it undoes ("o") is gone.
func renamedBack(nodes, op *pyjson.Object) (bool, error) {
	back, err := contains(nodes, get(op, "n"))
	if err != nil {
		return false, err
	}
	still, err := contains(nodes, get(op, "o"))
	return back && !still, err
}
