package readtools

import (
	"math"
	"regexp"
	"strconv"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// n4_schedule_audit (station-audits S12): every schedule below a base in one read-only call, with
// its out, next event, the special events (a calendar: its dates) that fall in the horizon, and
// whether anything reads it. It composes existing reads: one BQL list of schedule:ControlSchedule
// with each schedule's knob count, n4_read_schedule's decode per schedule (schedule.Summarize)
// and the station's date as n4_license_info reads it.

// Bounds of n4_schedule_audit.
const (
	SaDefaultMaxSchedules = 500
	saDefaultHorizon      = "7d"
	saMaxHorizonDays      = 366
	saTimeout             = 60
)

// saHorizonRe is a horizon: 1 to 3 digits of days.
var saHorizonRe = regexp.MustCompile(`^([1-9][0-9]{0,2})d$`)

// saRun is one audit: the station's day, the schedules read and the calendars resolved.
type saRun struct {
	t         *Tools
	def       schedule.Defaults
	today     time.Time
	dated     bool // today is the station's day
	days      int
	calendars map[string]*saCalendar
	notes     []any
}

// ScheduleAudit is n4_schedule_audit.
func (t *Tools) ScheduleAudit(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	slot, err := bql.BaseSlot(base)
	if err != nil {
		return nil, toolError("base must be a station slot ORD such as %s, got %s", pyjson.StrRepr(RootOrd),
			pyjson.StrRepr(base))
	}
	horizon := strArg(args, "horizon", saDefaultHorizon)
	m := saHorizonRe.FindStringSubmatch(horizon)
	days := 0
	if m != nil {
		days, _ = strconv.Atoi(m[1])
	}
	if days < 1 || days > saMaxHorizonDays {
		return nil, toolError("horizon must be a number of days such as 7d (1d to %dd), got %s", saMaxHorizonDays,
			pyjson.StrRepr(horizon))
	}
	start := t.Ctx.Clock()
	r := &saRun{t: t, def: saDefaults(t), days: days, calendars: map[string]*saCalendar{}, notes: []any{}}
	rows, knobs, cut, err := r.list(bql.StationSlot+slot, intArg(args, "max_schedules", SaDefaultMaxSchedules))
	if err != nil {
		return nil, err
	}
	if err := r.clock(); err != nil {
		return nil, err
	}
	read, unreadable, err := r.readAll(rows)
	if err != nil {
		return nil, err
	}
	out, err := r.reply(base, read, knobs)
	if err != nil {
		return nil, err
	}
	counts, _ := out.Get("counts")
	counts.(*pyjson.Object).Set("unreadable", len(unreadable))
	out.Set("unreadable", unreadable)
	out.Set("truncated", cut)
	out.Set("notes", r.notes)
	if r.dated {
		out.Set("today", r.today.Format("2006-01-02"))
	} else {
		out.Set("today", nil)
	}
	out.Set("horizon_days", days)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_schedule_audit", "base", args, len(rows))
	return out, nil
}

// saDefaults fills the slots a station leaves out from its type registry, as n4_read_schedule
// does; a client without one describes nothing (the slot is then not read, with a note).
func saDefaults(t *Tools) schedule.Defaults {
	tl, ok := t.Ctx.Session.Client.(plan.TypeLoader)
	return func(typeSpec, name string) (*pyjson.Object, error) {
		if !ok {
			return nil, nil
		}
		return plan.DeclaredDefault(tl, typeSpec, name)
	}
}

// clock reads the station's day; without it no special event is placed in the horizon.
func (r *saRun) clock() error {
	system, err := r.t.healthSystem()
	if err != nil {
		return err
	}
	c := stationClock(system)
	if !c.known {
		r.notes = append(r.notes, "upcoming special events not computed: "+c.why)
		return nil
	}
	r.today, r.dated = c.today, true
	return nil
}

// list is the schedules' slot paths below scope (at most max) and each one's knob count (nil
// when the station refused knobCount: then the fallback list is read and that is noted).
func (r *saRun) list(scope string, max int) ([]string, map[string]any, bool, error) {
	knobs := map[string]any{}
	fields := []string{"path", "knobs"}
	columns, raw, cut, _, err := r.t.query(scope, bql.ScheduleAuditQuery, max, saTimeout)
	if err != nil {
		section, fatal := r.t.sectionError(err)
		if fatal != nil {
			return nil, nil, false, fatal
		}
		r.notes = append(r.notes, "linked_targets not read (the station refused knobCount: "+
			pyjson.Str(lookup(section, "error"))+"); unlinked is not computed")
		fields, knobs = []string{"path"}, nil
		if columns, raw, cut, _, err = r.t.query(scope, bql.ScheduleListQuery, max, saTimeout); err != nil {
			return nil, nil, false, err
		}
	}
	rows, err := bql.ByPosition(columns, raw, fields)
	if err != nil {
		return nil, nil, false, err
	}
	if cut {
		r.notes = append(r.notes, "schedules cut at "+strconv.Itoa(len(rows))+": raise max_schedules or narrow base")
	}
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		p := cellText(row, "path")
		paths = append(paths, p)
		if knobs != nil {
			knobs[p] = saCount(cellText(row, "knobs"))
		}
	}
	return paths, knobs, cut, nil
}

// saCount is a knobCount cell as a number, nil when it is not a whole number.
func saCount(text string) any {
	if n, err := strconv.Atoi(text); err == nil && n >= 0 {
		return n
	}
	return nil
}
