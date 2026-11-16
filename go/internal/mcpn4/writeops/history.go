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
