package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// The reply of n4_schedule_audit: one entry per schedule read, the unlinked and no_next_event
// lists and the counts. A value that could not be read is null, with a note in that entry.

// reply builds the reply from the schedules read and their knob counts (nil: not read).
func (r *saRun) reply(base string, read []saRead, knobs map[string]any) (*pyjson.Object, error) {
	referenced := map[string]int{}
	entries := make([]*pyjson.Object, 0, len(read))
	for _, s := range read {
		e, err := r.entry(s, knobs, referenced)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	list, unlinked, noNext := []any{}, []any{}, []any{}
	calendars, upcoming := 0, 0
	for i, s := range read {
		e := entries[i]
		if s.sum.Kind.Name == "calendar" {
			calendars++
			e.Set("referenced_by", referenced[saKey(s.path)])
		} else if n, ok := lookup(e, "linked_targets").(int); ok && n == 0 {
			unlinked = append(unlinked, obj("schedule", s.ord))
		}
		if s.sum.NextTimeRead && s.sum.NextTime == nil {
			noNext = append(noNext, obj("schedule", s.ord))
		}
		if ev, _ := lookup(e, "special_events").(*pyjson.Object); ev != nil {
			if up, ok := lookup(ev, "upcoming").([]any); ok {
				upcoming += len(up)
			}
		}
		list = append(list, e)
	}
	var unlinkedOut, unlinkedCount any = unlinked, len(unlinked)
	if knobs == nil {
		unlinkedOut, unlinkedCount = nil, nil
	}
	return obj("base", base, "schedules", list, "unlinked", unlinkedOut, "no_next_event", noNext,
		"counts", obj("schedules", len(read), "calendars", calendars, "unlinked", unlinkedCount,
			"no_next_event", len(noNext), "upcoming_events", upcoming)), nil
}

// entry is {schedule, type, kind, out_value, next_time, next_value, special_events: {count,
// upcoming}, linked_targets, notes}, plus calendars for the references it holds.
func (r *saRun) entry(s saRead, knobs map[string]any, referenced map[string]int) (*pyjson.Object, error) {
	sum, notes := s.sum, []any{}
	for _, n := range sum.Notes {
		notes = append(notes, n)
	}
	e := obj("schedule", s.ord, "type", sum.Kind.Type, "kind", sum.Kind.Name,
		"out_value", saValue(sum.Out), "next_time", sum.NextTime, "next_value", saValue(sum.NextValue))
	if st := saField(sum.Out, "status"); st != nil && st != "ok" {
		e.Set("out_status", st)
	}
	events := obj("count", nil, "upcoming", nil)
	if sum.EventsRead {
		events.Set("count", len(sum.Events))
		up, refs, more, err := r.upcoming(sum.Events, referenced)
		if err != nil {
			return nil, err
		}
		if r.dated {
			events.Set("upcoming", up)
		}
		if len(refs) > 0 {
			e.Set("calendars", refs)
		}
		notes = append(notes, more...)
	}
	e.Set("special_events", events)
	if knobs != nil {
		linked := knobs[s.path]
		e.Set("linked_targets", linked)
		if linked == nil {
			notes = append(notes, "linked_targets not read: the knobCount cell is not a whole number")
		}
	} else {
		e.Set("linked_targets", nil)
	}
	e.Set("notes", notes)
	return e, nil
}

// upcoming is the events whose first day in the horizon is known ({name, kind, date, rule[,
// ranges][, calendar]}), the calendars the references name ({ord, resolves[, why]}) and the
// notes of events it could not place. Without the station's day it still resolves references.
func (r *saRun) upcoming(events []*pyjson.Object, referenced map[string]int) ([]any, []any, []any, error) {
	up, refs, notes := []any{}, []any{}, []any{}
	for _, ev := range events {
		rule, _ := lookup(ev, "rule").(*pyjson.Object)
		name, kind := pyjson.Str(lookup(ev, "name")), pyjson.Str(lookup(ev, "kind"))
		rules, where := []*pyjson.Object{ev}, ""
		if kind == "reference" {
			ref, _ := saField(rule, "calendar").(string)
			if ref == "" {
				notes = append(notes, "special event "+pyjson.StrRepr(name)+" references no calendar")
				continue
			}
			c, err := r.calendar(ref)
			if err != nil { // an authentication failure fails the call
				return nil, nil, nil, err
			}
			referenced[saKey(ref)]++
			refs = append(refs, saRef(c))
			if !c.resolves || c.why != "" {
				continue
			}
			rules, where = c.dates, c.ord
		}
		if !r.dated {
			continue
		}
		day, ok := r.first(rules)
		switch {
		case !ok:
			notes = append(notes, "special event "+pyjson.StrRepr(name)+" ("+kind+") not placed in the horizon: "+
				"its rule uses a field this audit does not evaluate (a calendar week, a week before the last, "+
				"or a wildcard date range)")
		case day != "":
			u := obj("name", name, "kind", kind, "date", day, "rule", rule)
			if rg, has := ev.Get("ranges"); has {
				u.Set("ranges", rg)
			}
			if where != "" {
				u.Set("calendar", where)
			}
			up = append(up, u)
		}
	}
	return up, refs, notes, nil
}

// first is the earliest day in the horizon any rule matches ("" for none); ok false when a
// rule cannot be evaluated.
func (r *saRun) first(rules []*pyjson.Object) (string, bool) {
	best := ""
	for _, x := range rules {
		rule, _ := lookup(x, "rule").(*pyjson.Object)
		d, found, ok := schedule.FirstMatch(pyjson.Str(lookup(x, "kind")), rule, r.today, r.days)
		if !ok {
			return "", false
		}
		if s := d.Format("2006-01-02"); found && (best == "" || s < best) {
			best = s
		}
	}
	return best, true
}

func saRef(c *saCalendar) *pyjson.Object {
	o := obj("ord", c.ord, "resolves", c.resolves)
	if c.why != "" {
		o.Set("why", c.why)
	}
	return o
}

// saValue is the plain value of a {value, status}, nil when it was not read.
func saValue(v *pyjson.Object) any { return saField(v, "value") }

// saField is lookup that reads a missing object (nil) as holding nothing.
func saField(o *pyjson.Object, k string) any {
	if o == nil {
		return nil
	}
	return lookup(o, k)
}
