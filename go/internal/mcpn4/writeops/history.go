package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// AddHistory is n4_add_history: one interval history extension per point, created already
// configured (historyName, interval, enabled) by one add op per point, all in one batch under
// one confirmation token. The inverse removes every extension under the name the station
// assigned.
var AddHistory = plan.Impl{ScopeOrds: historyPoints, Plan: addHistoryPlan, Inverse: addHistoryInverse,
	Readback: addHistoryReadback}

// maxHistoryPoints bounds one batch: a failure at op k leaves k extensions to roll back.
const maxHistoryPoints = 50

// historyExtTypes are the interval extensions: only they carry the top-level interval slot.
var historyExtTypes = []string{"history:NumericIntervalHistoryExt", "history:BooleanIntervalHistoryExt",
	"history:EnumIntervalHistoryExt", "history:StringIntervalHistoryExt"}

type historyData struct {
	parents   []any // the point handles, in op order
	requested []historySpec
}

type historySpec struct {
	ord, name, typ, pattern, resolved, interval string
	enabled                                     bool
}

func historyPoints(args *pyjson.Object) []any {
	v, _ := args.Get("points")
	list, _ := v.([]any)
	return append([]any{}, list...)
}

func addHistoryPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	spec, points, err := historyArgs(args)
	if err != nil {
		return nil, err
	}
	ops, inverse, targets := []any{}, []any{}, pyjson.NewObject()
	data := historyData{}
	seen := map[string]string{}
	for _, ord := range points {
		h, nodes, err := plan.Handle(c, ord, 1)
		if err != nil {
			return nil, err
		}
		if box.Node(nodes, spec.name) != nil {
			return nil, plan.ToolError(ord + " already has a child named " + pyjson.StrRepr(spec.name) +
				": remove it first or pass another ext_name")
		}
		s := spec
		s.ord = ord
		if r, ok := ResolveHistoryName(spec.pattern, ord+"/"+spec.name); ok {
			if other, dup := seen[r]; dup {
				return nil, plan.ToolError("history name " + pyjson.StrRepr(r) + " would repeat on " + other +
					" and " + ord + ": use a history_name pattern that includes the device, e.g. " +
					"S%parent.parent.parent.name%_%parent.name%")
			}
			seen[r], s.resolved = ord, r
		}
		slots := []any{
			obj("nm", "p", "n", "historyName", "t", "baja:Format", "v", s.pattern),
			obj("nm", "p", "n", "interval", "t", "baja:RelTime", "v", s.interval),
			obj("nm", "p", "n", "enabled", "t", "baja:Boolean", "v", strconv.FormatBool(s.enabled)),
		}
		ops = append(ops, obj("nm", "a", "h", h, "n", s.name, "b", obj("nm", "p", "t", s.typ, "s", slots)))
		inverse = append(inverse, obj("nm", "v", "h", h, "n", s.name))
		targets.Set(ord, h)
		data.parents = append(data.parents, h)
		data.requested = append(data.requested, s)
	}
	notes := []any{"The station may rename on collision; the real inverse uses the assigned name.",
		"Capacity stays at the extension default (500 records, Roll): about 41 h at 5 min."}
	if len(seen) < len(points) {
		notes = append(notes, "history_name uses tokens other than %parent...name%: names were not checked for repeats.")
	}
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets, Data: data}, nil
}

// historyArgs validates everything that needs no station: the points, the extension name and
// type, the pattern, the interval and enabled.
func historyArgs(args *pyjson.Object) (historySpec, []string, error) {
	var s historySpec
	raw, _ := args.Get("points")
	list, _ := raw.([]any)
	if len(list) == 0 || len(list) > maxHistoryPoints {
		return s, nil, plan.ToolError("points must list 1 to " + strconv.Itoa(maxHistoryPoints) + " point ORDs")
	}
	points, dup := []string{}, map[string]bool{}
	for _, p := range list {
		ord, ok := p.(string)
		if !ok || ord == "" || dup[ord] {
			return s, nil, plan.ToolError("points must be distinct non-empty ORD strings, got " + pyjson.Repr(p))
		}
		dup[ord] = true
		points = append(points, ord)
	}
	name, _ := args.Get("ext_name")
	if name == nil {
		name = "NumericInterval"
	}
	var err error
	if s.name, err = plan.Name("ext_name", name); err != nil {
		return s, nil, err
	}
	s.typ = "history:NumericIntervalHistoryExt"
	if t, ok := args.Get("ext_type"); ok {
		s.typ, _ = t.(string)
	}
	if !historyType(s.typ) {
		return s, nil, plan.ToolError("ext_type must be one of " + joinStrings(historyExtTypes))
	}
	pattern, _ := args.Get("history_name")
	if s.pattern, _ = pattern.(string); s.pattern == "" {
		return s, nil, plan.ToolError("history_name must be a non-empty baja:Format pattern, e.g. " +
			"S%parent.parent.parent.name%_%parent.name%")
	}
	interval, _ := args.Get("interval_ms")
	iv, err := plan.Coerce("baja:RelTime", interval, "0")
	if err != nil {
		return s, nil, err
	}
	if s.interval = pyjson.Str(iv); s.interval == "0" {
		return s, nil, plan.ToolError("interval_ms must be at least 1")
	}
	s.enabled = true
	if v, ok := args.Get("enabled"); ok {
		s.enabled = pyjson.Truthy(v)
	}
	return s, points, nil
}

func historyType(t string) bool {
	for _, k := range historyExtTypes {
		if k == t {
			return true
		}
	}
	return false
}

func joinStrings(xs []string) string {
	out := ""
	for i, x := range xs {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

// addHistoryInverse removes each extension under the name its reply assigned ("nn"), else the
// requested one.
func addHistoryInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(historyData)
	out := []any{}
	for i := range replies {
		nn := get(p.Ops[i].(*pyjson.Object), "n")
		if first, ok := replies[i].([]any); ok && len(first) > 0 {
			if reply, ok := first[0].(*pyjson.Object); ok {
				if v, ok := reply.Get("nn"); ok {
					nn = v
				}
			}
		}
		out = append(out, obj("nm", "v", "h", d.parents[i], "n", nn))
	}
	return out, nil
}

func addHistoryReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(historyData)
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, s := range d.requested {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		want := obj("ord", s.ord, "name", nn, "type", s.typ, "historyName", s.pattern,
			"interval", s.interval, "enabled", s.enabled)
		requested = append(requested, want)
		got, err := observeHistory(c, s.ord, nn)
		if err != nil {
			return nil, nil, nil, "", err
		}
		observed = append(observed, got)
		if !box.PyEqual(got, want) {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

// observeHistory reads one extension back; a slot at its type default is filled from the type
// definition (plan.Resolve), so a requested default value still compares equal.
func observeHistory(c plan.Client, ord, nn string) (any, error) {
	nodes, err := c.LoadTree(ord, 3, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	n := box.Node(nodes, nn)
	if n == nil {
		return nil, nil
	}
	out := obj("ord", ord, "name", nn, "type", get(n, "t"))
	for _, f := range []struct{ slot, vt string }{{"historyName", "baja:Format"},
		{"interval", "baja:RelTime"}, {"enabled", "baja:Boolean"}} {
		path := nn + "/" + f.slot
		if nodes, err = plan.Resolve(c, nodes, path); err != nil {
			return nil, err
		}
		v, err := plan.Observe(nodes, path, f.vt)
		if err != nil {
			return nil, err
		}
		out.Set(f.slot, v)
	}
	return out, nil
}
