package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// The per-schedule reads of n4_schedule_audit: one load per schedule (schedule.LoadDepth, as
// n4_read_schedule) and one per calendar a reference names that the list did not hold.

// saRead is one schedule read: its slot path, ORD and summary.
type saRead struct {
	path, ord string
	sum       *schedule.Summary
}

// saCalendar is a calendar a reference names: whether it resolves (loads as a calendar), its
// dates, and why not.
type saCalendar struct {
	ord      string
	resolves bool
	dates    []*pyjson.Object
	why      string
}

// readAll loads and summarizes each listed schedule; one that cannot be read is listed under
// unreadable {schedule, error}; an authentication failure fails the call.
func (r *saRun) readAll(paths []string) ([]saRead, []any, error) {
	read, unreadable := []saRead{}, []any{}
	for _, p := range paths {
		ord := "station:|" + p
		sum, why, err := r.load(ord)
		if err != nil {
			return nil, nil, err
		}
		if sum == nil {
			unreadable = append(unreadable, obj("schedule", ord, "error", why))
			continue
		}
		read = append(read, saRead{p, ord, sum})
		if sum.Kind.Name == "calendar" {
			r.calendars[saKey(p)] = saCalendarOf(ord, sum)
		}
	}
	return read, unreadable, nil
}

// load is the summary of the schedule at ord, nil with why when it cannot be read.
func (r *saRun) load(ord string) (*schedule.Summary, string, error) {
	nodes, _, err := r.t.timedLoad(ord, schedule.LoadDepth)
	if err != nil {
		section, fatal := r.t.sectionError(err)
		if fatal != nil {
			return nil, "", fatal
		}
		return nil, pyjson.Str(lookup(section, "error")), nil
	}
	sum, err := schedule.Summarize(nodes, r.def)
	if err != nil {
		return nil, err.Error(), nil
	}
	return sum, "", nil
}

func saCalendarOf(ord string, sum *schedule.Summary) *saCalendar {
	c := &saCalendar{ord: ord, resolves: true, dates: sum.Events}
	if !sum.EventsRead {
		c.why = "its dates could not be read"
	}
	return c
}

// calendar resolves the calendar a reference names, loading it when the list did not hold it.
func (r *saRun) calendar(ref string) (*saCalendar, error) {
	key := saKey(ref)
	if c, ok := r.calendars[key]; ok {
		return c, nil
	}
	ord := "station:|" + key
	c := &saCalendar{ord: ord}
	sum, why, err := r.load(ord)
	switch {
	case err != nil:
		return nil, err
	case sum == nil:
		c.why = why
	case sum.Kind.Name != "calendar":
		c.why = "it is a " + sum.Kind.Type + ", not a schedule:CalendarSchedule"
	default:
		c = saCalendarOf(ord, sum)
	}
	r.calendars[key] = c
	return c, nil
}

// saKey is the station part of an ORD ("station:|slot:/A/Cal" and "slot:/A/Cal" are both
// "slot:/A/Cal"): the key calendars are kept and matched under.
func saKey(ord string) string {
	return ord[strings.LastIndex(ord, "|")+1:]
}
