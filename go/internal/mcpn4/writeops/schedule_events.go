package writeops

import (
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// today is the date special events are checked against (a var so the plan stays testable).
var today = time.Now

// eventParts replaces the specialEvents list with one set: the events kept in their order (the
// order is their priority), minus remove_special_event, plus add_special_event at the end.
func eventParts(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart {
	addRaw, adding := args.Get("add_special_event")
	removeRaw, removing := args.Get("remove_special_event")
	if !adding && !removing {
		return nil
	}
	if !s.kind.SpecialEvents {
		*f = append(*f, "special events: a "+s.kind.Name+" schedule has none")
		return nil
	}
	addList, ok := addRaw.([]any)
	if adding && !ok {
		*f = append(*f, "add_special_event: must be a list of {name, kind, rule, ranges}")
	}
	removeList, ok := removeRaw.([]any)
	if removing && !ok {
		*f = append(*f, "remove_special_event: must be a list of event names")
	}
	before := len(*f)
	allow, _ := args.Get("allow_expired")
	facets, _ := s.model.Get("facets")
	added := schedule.SpecialEvents(addList, s.kind, pyjson.Str(facets), today(), allow == true, f)
	path := "schedule/specialEvents"
	old := box.Node(s.nodes, path)
	kept, names := keptEvents(s, path, removeList, f)
	for _, e := range added {
		if names[e.Name] {
			*f = append(*f, "add_special_event: "+pyjson.StrRepr(e.Name)+" already exists; remove it in the same call to replace it")
		}
		names[e.Name] = true
		if e.Kind == "reference" {
			checkCalendar(s, e.Where+".rule.calendar", pyjson.Str(get(e.Rule, "calendar")), f)
		}
	}
	if len(*f) > before || old == nil { // nothing is encoded from a request with faults
		return nil
	}
	removed := len(box.Children(s.nodes, path).Keys()) - (len(kept))
	body := schedule.Strip(old)
	want := []any{}
	for _, n := range kept {
		want = append(want, eventView(s.model, n))
	}
	for _, e := range added {
		kept = append(kept, schedule.EventBody(s.kind, e))
		views := []any{}
		for _, r := range e.Ranges {
			views = append(views, schedule.View(s.kind, r))
		}
		want = append(want, obj("name", e.Name, "kind", e.Kind, "rule", schedule.RuleView(e.Kind, e.Rule), "ranges", views))
	}
	newBody := obj("nm", "p", "t", get(body, "t"), "s", kept)
	h, _ := box.Node(s.nodes, "schedule").Get("h")
	return []schedulePart{{op: setOp(h, "specialEvents", newBody), inverse: setOp(h, "specialEvents", body),
		target: "schedule", key: "special_events", want: want,
		note: itoa(len(added)) + " event(s) added, " + itoa(removed) + " removed; the order is the priority"}}
}

// keptEvents is the stripped bodies of the events not removed, in order, and the names of all
// of them; a name to remove that no event has is a fault.
func keptEvents(s *scheduleState, path string, remove []any, f *schedule.Faults) ([]any, map[string]bool) {
	drop := map[string]bool{}
	for _, r := range remove {
		name, _ := r.(string)
		if drop[name] {
			*f = append(*f, "remove_special_event: "+pyjson.Repr(r)+" is named twice")
		}
		drop[name] = true
	}
	kept, names := []any{}, map[string]bool{}
	for _, slot := range box.Children(s.nodes, path).Keys() {
		name := bql.UnescapeSlot(slot)
		if drop[name] {
			delete(drop, name)
			continue
		}
		names[name] = true
		body := schedule.Strip(box.Node(s.nodes, path+"/"+slot))
		body.Set("n", slot)
		kept = append(kept, body)
	}
	for _, r := range remove {
		if name, _ := r.(string); drop[name] {
			*f = append(*f, "remove_special_event: no event is named "+pyjson.Repr(r))
			delete(drop, name)
		}
	}
	return kept, names
}

// eventView is a kept event as the read-back compares it: its name, rule and ranges.
func eventView(model *pyjson.Object, body any) *pyjson.Object {
	name := bql.UnescapeSlot(pyjson.Str(get(body.(*pyjson.Object), "n")))
	events, _ := model.Get("special_events")
	list, _ := events.([]any)
	for _, e := range list {
		if o := e.(*pyjson.Object); get(o, "name") == name {
			return obj("name", name, "kind", get(o, "kind"), "rule", get(o, "rule"), "ranges", get(o, "ranges"))
		}
	}
	return obj("name", name)
}

// checkCalendar refuses a reference whose ORD does not name a calendar (a read only; the
// calendar need not be inside the write scope). An ORD that is not a slot ORD already has its
// format fault, so it is not loaded.
func checkCalendar(s *scheduleState, where, ord string, f *schedule.Faults) {
	if !strings.HasPrefix(ord, "station:|slot:/") {
		return
	}
	nodes, err := s.c.LoadTree(ord, 1, box.LoadOptions{})
	head := where + ": " + pyjson.StrRepr(ord)
	switch {
	case err != nil:
		*f = append(*f, head+" names no component ("+err.Error()+")")
	case get(box.Node(nodes, ""), "t") != "schedule:CalendarSchedule":
		*f = append(*f, head+" names no calendar (it is a "+pyjson.Str(get(box.Node(nodes, ""), "t"))+
			"); n4_read_schedule a calendar's ORD to check it")
	}
}
