package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// datesPart removes and adds the dates of a calendar (or of a trigger's dates calendar). A date
// is a dynamic child of the calendar: each removal is one remove, each addition one add of the
// bare date type followed by one set per frozen child by path (yearSchedule, monthSchedule, ...),
// the add-then-set-by-path pattern of n4_add_alarm, so no add carries nested components. The
// inverse undoes the additions, then re-adds each removed date the same way from its loaded
// children.
func datesPart(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart {
	addRaw, adding := args.Get("add_date")
	removeRaw, removing := args.Get("remove_date")
	if !adding && !removing {
		return nil
	}
	if !s.kind.Dates {
		*f = append(*f, "dates: a "+s.kind.Name+" schedule has none (its special events can reference a calendar)")
		return nil
	}
	addList, ok := addRaw.([]any)
	if adding && !ok {
		*f = append(*f, "add_date: must be a list of {name, kind, rule}")
	}
	removeList, ok := removeRaw.([]any)
	if removing && !ok {
		*f = append(*f, "remove_date: must be a list of date names")
	}
	path := ""
	if s.kind.Times {
		path = "dates"
	}
	allow, _ := args.Get("allow_expired")
	added := schedule.Dates(addList, today(), allow == true, f)
	h, _ := box.Node(s.nodes, path).Get("h")
	slots, kept := dateSlots(s, path, removeList, f)
	var ops, inverse []any
	want := []any{}
	for _, slot := range slots.removed {
		ops = append(ops, obj("nm", "v", "h", h, "n", slot))
	}
	for _, slot := range kept {
		want = append(want, dateView(s.model, bql.UnescapeSlot(slot)))
	}
	for _, d := range added {
		switch {
		case slots.names[d.Name]:
			*f = append(*f, "add_date: "+pyjson.StrRepr(d.Name)+" already exists")
		case removedName(slots.removed, d.Name): // the rollback checks each op against the end state
			*f = append(*f, "add_date: "+pyjson.StrRepr(d.Name)+" is also removed in this call; replace a date in "+
				"two calls (remove_date, then add_date) so each one rolls back on its own")
		}
		slots.names[d.Name] = true
		ops = append(ops, addByPath(h, d.Slot, d.Body)...)
		inverse = append(inverse, obj("nm", "v", "h", h, "n", d.Slot))
		want = append(want, obj("name", d.Name, "kind", d.Kind, "rule", schedule.RuleView(d.Kind, d.Rule)))
	}
	for _, slot := range slots.removed {
		inverse = append(inverse, addByPath(h, slot, box.Node(s.nodes, joinPath(path, slot)))...)
	}
	if len(ops) == 0 { // empty add_date and remove_date lists change nothing
		return nil
	}
	return []schedulePart{{op: ops, inverse: inverse, target: path, key: "dates", want: want,
		note: fmt.Sprintf("%d date(s) added, %d removed", len(added), len(slots.removed))}}
}

// dateSet is the slots of the dates to remove and the names of the dates left.
type dateSet struct {
	removed []string
	names   map[string]bool
}

// dateSlots splits the container's dates into those removed (by shown name) and those kept, in
// order; a name to remove that no date has is a fault.
func dateSlots(s *scheduleState, path string, remove []any, f *schedule.Faults) (dateSet, []string) {
	drop := map[string]bool{}
	for _, r := range remove {
		name, _ := r.(string)
		drop[name] = true
	}
	set, kept := dateSet{names: map[string]bool{}}, []string{}
	for _, slot := range box.Children(s.nodes, path).Keys() {
		if !isDateType(get(box.Node(s.nodes, joinPath(path, slot)), "t")) {
			continue
		}
		if name := bql.UnescapeSlot(slot); drop[name] {
			delete(drop, name)
			set.removed = append(set.removed, slot)
		} else {
			set.names[name] = true
			kept = append(kept, slot)
		}
	}
	for _, r := range remove {
		if name, _ := r.(string); drop[name] {
			*f = append(*f, "remove_date: no date is named "+pyjson.Repr(r))
			delete(drop, name)
		}
	}
	return set, kept
}

// addByPath is the add of body's type bare under h as slot, then a set of each child of body by
// path (a date's children are all frozen slots its add creates).
func addByPath(h any, slot string, body *pyjson.Object) []any {
	ops := []any{obj("nm", "a", "h", h, "n", slot, "b", obj("nm", "p", "t", get(body, "t")))}
	kids, _ := get(body, "s").([]any)
	for _, k := range kids {
		kid, ok := k.(*pyjson.Object)
		if !ok {
			continue
		}
		ops = append(ops, setOp(h, slot+"/"+pyjson.Str(get(kid, "n")), schedule.Strip(kid)))
	}
	return ops
}

func isDateType(t any) bool {
	for _, e := range schedule.EventKinds {
		if e.Type == t && e.Name != "reference" {
			return true
		}
	}
	return false
}

// dateView is a kept date as n4_read_schedule shows it.
func dateView(model *pyjson.Object, name string) any {
	dates, _ := model.Get("dates")
	list, _ := dates.([]any)
	for _, d := range list {
		if o := d.(*pyjson.Object); get(o, "name") == name {
			return o
		}
	}
	return obj("name", name)
}

func removedName(slots []string, name string) bool {
	for _, slot := range slots {
		if bql.UnescapeSlot(slot) == name {
			return true
		}
	}
	return false
}
