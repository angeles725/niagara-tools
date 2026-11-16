package writeops

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// AddHistory is n4_add_history: one interval or change-of-value history extension per point, of
// the type that fits the point (numeric, boolean, enum or string), created already configured
// (historyName, interval for an interval extension, changeTolerance for a numeric COV one when
// asked, enabled) by one add op per point and, when a capacity or a full policy is asked, a set
// op for each right after it, all in one batch under one confirmation token. A station failure at
// op k stops the batch: the ops before it stand, the batch is journaled in-doubt with their
// inverse, and n4_rollback removes the extensions it added. The dry run predicts each history id ("/<station>/<name>") and refuses one that would repeat, in
// the batch or against the station's extensions. The inverse removes every extension under the
// name the station assigned.
var AddHistory = plan.Impl{ScopeOrds: historyPoints, PlanWith: addHistoryPlan, Inverse: addHistoryInverse,
	Readback: addHistoryReadback}

// maxHistoryPoints bounds one batch: a failure at op k leaves k extensions to roll back.
const maxHistoryPoints = 50

// historyExtTypes are the interval extensions (they alone carry the top-level interval slot)
// and the change-of-value ones (history-rt ext/B<Kind>CovHistoryExt).
var historyExtTypes = []string{"history:NumericIntervalHistoryExt", "history:BooleanIntervalHistoryExt",
	"history:EnumIntervalHistoryExt", "history:StringIntervalHistoryExt", "history:NumericCovHistoryExt",
	"history:BooleanCovHistoryExt", "history:EnumCovHistoryExt", "history:StringCovHistoryExt"}

type historyData struct {
	parents   []any // the point handles, in op order
	adds      []int // the index of each point's add op among the ops
	requested []historySpec
}

// historySpec is one planned extension: its point, slot name, type, history name pattern (and
// the name and id it resolves to, "" when the pattern cannot be resolved from the ORD), mode
// (Interval or Cov), interval ("" for Cov), capacity (history:Capacity encoded, "" for the
// default), its record count ("" unless given as one), the full policy tag ("" for the default),
// the change tolerance (nil unless given) and enabled.
type historySpec struct {
	ord, name, typ, pattern, resolved, id, mode, interval, capacity, records, fullPolicy string
	tolerance                                                                            any
	enabled                                                                              bool
}

func historyPoints(args *pyjson.Object) []any {
	v, _ := args.Get("points")
	list, _ := v.([]any)
	return append([]any{}, list...)
}

func addHistoryPlan(c plan.Client, args *pyjson.Object, _ *plan.State, sess *session.Session) (*plan.Planned, error) {
	spec, points, err := historyArgs(args)
	if err != nil {
		return nil, err
	}
	ops, inverse, targets := []any{}, []any{}, pyjson.NewObject()
	data, kinds := historyData{}, newPointKinds(c)
	seen := map[string]string{}
	for _, ord := range points {
		h, nodes, err := plan.Handle(c, ord, 1)
		if err != nil {
			return nil, err
		}
		s, err := spec.forPoint(kinds, ord, get(box.Node(nodes, ""), "t"))
		if err != nil {
			return nil, err
		}
		if box.Node(nodes, s.name) != nil {
			return nil, plan.ToolError(ord + " already has a child named " + pyjson.StrRepr(s.name) +
				": remove it first or pass another ext_name")
		}
		if r, ok := ResolveHistoryName(s.pattern, ord+"/"+s.name); ok {
			if other, dup := seen[r]; dup {
				return nil, plan.ToolError(fmt.Sprintf(msgHistoryNameRepeat, pyjson.StrRepr(r), other, ord))
			}
			seen[r], s.resolved, s.id = ord, r, "/"+sess.StationName+"/"+r
		}
		data.adds = append(data.adds, len(ops))
		ops = append(ops, obj("nm", "a", "h", h, "n", s.name, "b", obj("nm", "p", "t", s.typ, "s", historySlots(s))))
		if s.capacity != "" {
			ops = append(ops, capacityOp(h, s.name, s.capacity))
		}
		if s.fullPolicy != "" {
			ops = append(ops, fullPolicyOp(h, s.name, s.fullPolicy))
		}
		inverse = append(inverse, obj("nm", "v", "h", h, "n", s.name))
		targets.Set(ord, h)
		data.parents = append(data.parents, h)
		data.requested = append(data.requested, s)
	}
	cut, err := checkStationIDs(c, data.requested)
	if err != nil {
		return nil, err
	}
	notes := []any{noteHistoryRename, capacityNote(spec)}
	if len(seen) < len(points) {
		notes = append(notes, noteHistoryUnchecked)
	}
	if cut {
		notes = append(notes, noteHistoryIDsCut)
	}
	if spec.capacity != "" {
		notes = append(notes, noteHistoryCapacityOp)
	}
	if spec.fullPolicy != "" {
		notes = append(notes, noteHistoryFullPolicyOp)
	}
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets, Data: data}, nil
}

// historySlots are the configured slots of the add body; a change-of-value extension has no
// interval.
func historySlots(s historySpec) []any {
	slots := []any{obj("nm", "p", "n", "historyName", "t", "baja:Format", "v", s.pattern)}
	if s.mode == "Interval" {
		slots = append(slots, obj("nm", "p", "n", "interval", "t", "baja:RelTime", "v", s.interval))
	}
	slots = append(slots, obj("nm", "p", "n", "enabled", "t", "baja:Boolean", "v", strconv.FormatBool(s.enabled)))
	if f, ok := s.tolerance.(float64); ok {
		slots = append(slots, toleranceSlot(f))
	}
	return slots
}
