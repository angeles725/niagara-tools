package writeops

import (
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

// registerTool is n4_register_bacnet_spyder: its add is a create, so its rollback removes the device
// under the same guard.
const registerTool = "n4_register_bacnet_spyder"

// viewsJournal is the journal's merged batch views, in journal order (journal.Journal.Views).
type viewsJournal interface {
	Views() ([]*pyjson.Object, error)
}

// laterWork refuses the rollback of create batch bid (its view, its single remove op and its
// recorded targets) when later work sits under the component it created; any other batch passes.
// With chain, the later batches that stay under the component are no refusal: they are returned
// (newest first) to be removed with it; a child or link no batch made still refuses.
func laterWork(c plan.Client, w *plan.State, view *pyjson.Object, ops []any, targets *pyjson.Object, bid string,
	chain bool) ([]string, error) {
	if tool := get(view, "tool"); tool != createTool && tool != registerTool || len(ops) != 1 {
		return nil, nil
	}
	remove := ops[0].(*pyjson.Object)
	created, err := childOrd(ordOf(targets, get(remove, "h")), get(remove, "n"))
	if err != nil {
		return nil, err
	}
	later, err := laterBatchViews(w, view, bid, created)
	if err != nil {
		return nil, err
	}
	batches := batchLabels(later)
	made, err := extraChildren(c, view, ordOf(targets, get(remove, "h")), get(remove, "n").(string))
	if err != nil {
		return nil, err
	}
	extras := make([]string, 0, len(made))
	for _, x := range made {
		extras = append(extras, x.name)
	}
	if chain && len(later) > 0 {
		if err := chainStaysUnder(later, created); err != nil {
			return nil, err
		}
		if extras = uncovered(made, later, created); len(extras) == 0 {
			return batches, nil
		}
	}
	if len(batches) == 0 && len(extras) == 0 {
		return nil, nil
	}
	return nil, plan.ToolError(laterWorkRefusal(bid, created, batches, extras, chain))
}

// laterBatches is "<batch id> (<tool>)" of every batch journaled after bid on the same station,
// newest first, whose recorded targets are created or under it; a rollback batch, and a batch
// undone by a completed rollback that did not fail, are no later work.
func laterBatches(w *plan.State, view *pyjson.Object, bid, created string) ([]string, error) {
	later, err := laterBatchViews(w, view, bid, created)
	return batchLabels(later), err
}

func batchLabels(views []*pyjson.Object) []string {
	var out []string
	for _, v := range views {
		out = append(out, pyjson.Str(get(v, "batch_id"))+" ("+pyjson.Str(get(v, "tool"))+")")
	}
	return out
}

// laterBatchViews is the views behind laterBatches, newest first.
func laterBatchViews(w *plan.State, view *pyjson.Object, bid, created string) ([]*pyjson.Object, error) {
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
	vanishedWith(views, undone) // what a chain=true rollback removed with its create (#971 item 2)
	var out []*pyjson.Object
	for i := len(views) - 1; i > after && after >= 0; i-- {
		v := views[i]
		id, _ := get(v, "batch_id").(string)
		if pyjson.Truthy(get(v, "rollback_of")) || undone[id] || !box.PyEqual(get(v, "station_name"), get(view, "station_name")) {
			continue
		}
		if t, ok := get(v, "targets").(*pyjson.Object); ok && anyUnder(under, t.Keys()) {
			out = append(out, v)
		}
	}
	return out, nil
}

// chainStaysUnder refuses a chain when a later batch also wrote outside the created component:
// its inverse would restore something the remove does not take with it.
func chainStaysUnder(later []*pyjson.Object, created string) error {
	under, err := ord.NewScope([]string{created})
	if err != nil {
		return err
	}
	for _, v := range later {
		if t, ok := get(v, "targets").(*pyjson.Object); ok {
			for _, o := range t.Keys() {
				if under.Check(o) != nil {
					return plan.ToolError(chainOutsideRefusal(pyjson.Str(get(v, "batch_id"))+" ("+pyjson.Str(get(v, "tool"))+")",
						o, created))
				}
			}
		}
	}
	return nil
}

// uncovered is the extra children no later chain batch made: a component one of them added under
// created, or a link a later n4_create_link batch made on created, matched by the link name its inverse
// removes. A link added by hand beside a batch-made one stays uncovered, so chain refuses it.
func uncovered(extras []extraChild, later []*pyjson.Object, created string) []string {
	added, links := map[string]bool{}, map[string]bool{}
	for _, v := range later {
		t, _ := get(v, "targets").(*pyjson.Object)
		if t == nil {
			continue
		}
		if get(v, "tool") == "n4_create_link" {
			inv, _ := get(v, "inverse").([]any)
			for _, e := range inv {
				if op, ok := e.(*pyjson.Object); ok && get(op, "nm") == "v" && ordOf(t, get(op, "h")) == created {
					if n, ok := get(op, "n").(string); ok {
						links[n] = true
					}
				}
			}
		}
		ops, _ := get(v, "ops").([]any)
		for _, e := range ops {
			if op, ok := e.(*pyjson.Object); ok && get(op, "nm") == "a" && ordOf(t, get(op, "h")) == created {
				if n, ok := get(op, "n").(string); ok {
					added[n] = true
				}
			}
		}
	}
	var out []string
	for _, x := range extras {
		if !(added[x.name] || x.link && links[x.name]) {
			out = append(out, x.name)
		}
	}
	return out
}

func anyUnder(scope *ord.Scope, ords []string) bool {
	for _, o := range ords {
		if scope.Check(o) == nil {
			return true
		}
	}
	return false
}
