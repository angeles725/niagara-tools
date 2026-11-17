package schedule

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Summary is what the schedule audit reads of one schedule: its kind, out, next time and value,
// and its special events (a calendar: its dates), each event {name, kind, rule, ranges}. Unlike
// Read it never fails on one slot: a slot it cannot read is nil and Notes says why.
type Summary struct {
	Kind                     *Kind
	Out, NextValue           *pyjson.Object // {value, status}
	NextTime                 any            // the BAbsTime text, nil for the null time or when unreadable
	NextTimeRead, EventsRead bool
	Events                   []*pyjson.Object
	Notes                    []string
}

// Summarize reads the Summary of a schedule loaded to LoadDepth; an error only when the
// component is no schedule kind.
func Summarize(nodes *pyjson.Object, def Defaults) (*Summary, error) {
	t := &tree{nodes: nodes, def: def}
	k := KindOf(t.typeOf(""))
	if k == nil {
		return nil, fmt.Errorf("%s is not a schedule kind", t.typeOf(""))
	}
	s := &Summary{Kind: k}
	if k.ValueType != "" {
		s.Out = s.status(t, "out")
		s.NextValue = s.status(t, "nextValue")
		if v, err := t.simple("", "nextTime"); err != nil {
			s.note("next_time", err)
		} else {
			s.NextTime, s.NextTimeRead = absTime(v), true
		}
	}
	model := pyjson.NewObject()
	var err error
	key := "special_events"
	if k.Dates {
		key = "dates"
		err = t.dates(k, model)
	} else {
		err = t.events(k, model)
	}
	if err != nil {
		s.note(key, err)
		return s, nil
	}
	list, _ := model.Get(key)
	for _, e := range list.([]any) {
		s.Events = append(s.Events, e.(*pyjson.Object))
	}
	s.EventsRead = true
	return s, nil
}

func (s *Summary) status(t *tree, slot string) *pyjson.Object {
	v, err := t.status("", slot)
	if err != nil {
		s.note(slot, err)
		return nil
	}
	return v
}

func (s *Summary) note(what string, err error) {
	s.Notes = append(s.Notes, what+" not read: "+err.Error())
}
