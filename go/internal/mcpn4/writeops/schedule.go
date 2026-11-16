package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// SetSchedule is n4_set_schedule: the days given of the weekly schedule and the default output in
// one batch.
// Each part is one set op that replaces a whole subtree, so its inverse is one set op with the
// subtree as loaded: the rollback restores the previous schedule exactly.
var SetSchedule = plan.Impl{ScopeOrds: ordArg, Plan: schedulePlan, Readback: scheduleReadback}

// scheduleState is a schedule loaded for a write: its handle, nodes, kind and decoded model.
type scheduleState struct {
	h     any
	nodes *pyjson.Object
	kind  *schedule.Kind
	model *pyjson.Object
	def   schedule.Defaults
	c     plan.Client // for read-only checks of what a request names (a referenced calendar)
}

// loadSchedule loads and decodes the schedule at ord (the station's type registry fills the
// slots it leaves out).
func loadSchedule(c plan.Client, ord string) (*scheduleState, error) {
	h, nodes, err := plan.Handle(c, ord, schedule.LoadDepth)
	if err != nil {
		return nil, err
	}
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, plan.ToolError("this client cannot read the station's type registry, which n4_set_schedule needs")
	}
	def := func(typeSpec, name string) (*pyjson.Object, error) { return plan.DeclaredDefault(tl, typeSpec, name) }
	model, err := schedule.Read(nodes, def)
	if err != nil {
		return nil, plan.ToolError(ord + ": " + err.Error())
	}
	t, _ := box.Node(nodes, "").Get("t")
	return &scheduleState{h, nodes, schedule.KindOf(pyjson.Str(t)), model, def, c}, nil
}

// schedulePart is one planned part: its op and inverse (each one op or a list), what the
// read-back expects under key, and a note.
type schedulePart struct {
	op, inverse any
	target      string // the ORD of the component the op is sent to, below the schedule
	key         string
	want        any
	note        string
}

// scheduleParts are the planners of each argument, in plan order.
var scheduleParts = []func(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart{
	weeklyParts, defaultOutputPart, eventParts, datesPart, timesPart,
}

func schedulePlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	ordText := str(args, "ord")
	s, err := loadSchedule(c, ordText)
	if err != nil {
		return nil, unreadSchedule(ordText, args, err) // schedule_precheck.go
	}
	var faults schedule.Faults
	var parts []schedulePart
	for _, planner := range scheduleParts {
		parts = append(parts, planner(s, args, &faults)...)
	}
	if len(parts) == 0 && len(faults) == 0 {
		faults = append(faults, "nothing to set: give weekly, default_output, add_special_event, remove_special_event, "+
			"add_date, remove_date or times")
	}
	if err := faults.Err(); err != nil {
		return nil, plan.ToolError(err.Error())
	}
	p := &plan.Planned{Targets: pyjson.NewObject()}
	p.Targets.Set(ordText, s.h)
	want := pyjson.NewObject()
	for _, part := range parts {
		ops, inverse := opsOf(part.op), opsOf(part.inverse) // a part sends one op or several
		if len(ops) == 0 {
			continue
		}
		if h, _ := ops[0].(*pyjson.Object).Get("h"); part.target != "" {
			p.Targets.Set(ordText+"/"+part.target, h)
		}
		p.Ops, p.Inverse = append(p.Ops, ops...), append(p.Inverse, inverse...)
		if part.note != "" {
			p.Notes = append(p.Notes, part.note)
		}
		putWant(want, part.key, part.want)
	}
	p.Data = want
	return p, nil
}

// weeklyParts replaces the day subtree of each day given (and only those).
func weeklyParts(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart {
	raw, given := args.Get("weekly")
	if !given {
		return nil
	}
	if !s.kind.Weekly {
		*f = append(*f, "weekly: a "+s.kind.Name+" schedule has no weekly schedule")
		return nil
	}
	arg, ok := raw.(*pyjson.Object)
	if !ok {
		*f = append(*f, "weekly: must be an object {day: [{start, end, value}]}")
		return nil
	}
	facets, _ := s.model.Get("facets")
	before := len(*f)
	days := schedule.Weekly(arg, s.kind, pyjson.Str(facets), f)
	if len(*f) > before { // nothing is encoded from a request with faults
		return nil
	}
	var parts []schedulePart
	for _, day := range schedule.Days {
		ranges, ok := days[day]
		if !ok {
			continue
		}
		daily := "schedule/week/" + day
		h, _ := box.Node(s.nodes, daily).Get("h")
		views := []any{}
		for _, r := range ranges {
			views = append(views, schedule.View(s.kind, r))
		}
		parts = append(parts, schedulePart{
			op:      setOp(h, "day", schedule.DayBody(s.kind, ranges)),
			inverse: setOp(h, "day", schedule.Strip(box.Node(s.nodes, daily+"/day"))),
			target:  daily, key: "weekly." + day, want: views,
			note: "replaces " + day + " (" + itoa(len(ranges)) + " range(s)); the days not given keep their ranges"})
	}
	return parts
}

// defaultOutputPart sets the default output (null: the type default with a null status).
func defaultOutputPart(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart {
	value, given := args.Get("default_output")
	if !given {
		return nil
	}
	if s.kind.ValueType == "" {
		*f = append(*f, "default_output: a "+s.kind.Name+" schedule has no default output")
		return nil
	}
	facets, _ := s.model.Get("facets")
	if msg := schedule.CheckValue(s.kind, pyjson.Str(facets), value, true); msg != "" {
		*f = append(*f, "default_output: "+msg)
		return nil
	}
	old, err := previousSlot(s, "defaultOutput")
	if err != nil {
		*f = append(*f, "default_output: "+err.Error())
		return nil
	}
	return []schedulePart{{op: setOp(s.h, "defaultOutput", schedule.StatusBody(s.kind, value)),
		inverse: setOp(s.h, "defaultOutput", old), key: "default_output", want: schedule.StatusView(s.kind, value)}}
}

// previousSlot is the body that restores slot of the schedule: as loaded, or its declared
// default when the station left it out (it held that default; loadSchedule's schedule.Read has
// already refused a left-out slot the type registry does not describe).
func previousSlot(s *scheduleState, slot string) (*pyjson.Object, error) {
	if n := box.Node(s.nodes, slot); n != nil {
		return schedule.Strip(n), nil
	}
	d, err := s.def(s.kind.Type, slot)
	if err != nil {
		return nil, err
	}
	return schedule.Strip(d), nil
}

func itoa(n int) string { return strconv.Itoa(n) }
