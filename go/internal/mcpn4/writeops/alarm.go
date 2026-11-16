package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// AddAlarm is n4_add_alarm: one alarm extension per point (alarm:AlarmSourceExt with the
// offnormal and fault algorithms of its kind, as the alarm palette presets them), configured
// (class, texts, delays, enables, limits or alarm values) in one batch under one confirmation
// token. The kind follows the point type unless given; the dry run refuses a kind that does not
// fit, a class the station does not hold (one BQL read) and a second extension of the same kind
// on a point unless allow_second. The inverse removes every extension; the read-back reads every
// slot it set and reports each value the station stored differently.
var AddAlarm = plan.Impl{ScopeOrds: historyPoints, PlanWith: addAlarmPlan, Inverse: addAlarmInverse,
	Readback: addAlarmReadback}

type alarmData struct {
	parents []any // the point handles, in op order
	adds    []int // the index of each point's add op among the ops
	points  []alarmPoint
}

// alarmPoint is one planned extension: its point, requested name, kind and every slot written.
type alarmPoint struct {
	ord, name string
	kind      alarmKind
	writes    []alarmWrite
}

func addAlarmPlan(c plan.Client, args *pyjson.Object, _ *plan.State, _ *session.Session) (*plan.Planned, error) {
	spec, err := alarmArgs(args)
	if err != nil {
		return nil, err
	}
	unchecked, err := checkAlarmClass(c, spec.class)
	if err != nil {
		return nil, err
	}
	ops, inverse, targets := []any{}, []any{}, pyjson.NewObject()
	data, kinds, noLimits := alarmData{}, newPointKinds(c), false
	for _, ord := range spec.points {
		h, nodes, err := plan.Handle(c, ord, 2)
		if err != nil {
			return nil, err
		}
		pt, err := spec.forPoint(kinds, ord, nodes)
		if err != nil {
			return nil, err
		}
		noLimits = noLimits || (pt.kind.name == "out_of_range" && len(spec.paramWrites["high_limit"])+
			len(spec.paramWrites["low_limit"]) == 0)
		data.adds = append(data.adds, len(ops))
		ops = append(ops, alarmOps(h, pt, spec.extInAdd)...)
		inverse = append(inverse, obj("nm", "v", "h", h, "n", pt.name))
		targets.Set(ord, h)
		data.parents = append(data.parents, h)
		data.points = append(data.points, pt)
	}
	notes := []any{noteAlarmRename, noteAlarmAlgorithm}
	if noLimits {
		notes = append(notes, noteAlarmNoLimits)
	}
	if unchecked {
		notes = append(notes, noteAlarmClassesCut)
	}
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: targets, Data: data}, nil
}

// forPoint is the extension planned for the point at ord: the kind (given, else the default of
// the point's family) must fit the point and take every algorithm argument given; the name must
// be free; a same-kind extension already there refuses unless allow_second.
func (s alarmSpec) forPoint(k *pointKinds, ord string, nodes *pyjson.Object) (alarmPoint, error) {
	pointType := get(box.Node(nodes, ""), "t")
	spec, _ := pointType.(string)
	family, err := k.kind(spec)
	if err != nil {
		return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmNoRegistry, ord, err.Error()))
	}
	if family == "" {
		return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmNotAPoint, ord, pyjson.Str(pointType)))
	}
	name := s.kind
	if name == "" {
		name = alarmDefaultKind[family]
	}
	kind, _ := findAlarmKind(name)
	if !kind.fits(family) {
		return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmKindMismatch, kind.name, ord, strings.ToLower(family),
			spec, strings.Join(fittingAlarmKinds(family), ", ")))
	}
	pt := alarmPoint{ord: ord, name: s.name, kind: kind, writes: append([]alarmWrite{}, s.ext...)}
	for _, p := range s.params {
		if !kind.takes(p) {
			return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmParam, p, kind.name, ord, kindTaking(p)))
		}
		pt.writes = append(pt.writes, s.paramWrites[p]...)
	}
	if pt.name == "" {
		pt.name = kind.palette
	}
	if box.Node(nodes, pt.name) != nil {
		return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmChild, ord, pyjson.StrRepr(pt.name)))
	}
	if other := sameKindExt(nodes, kind); other != "" && !s.allowSecond {
		return alarmPoint{}, plan.ToolError(fmt.Sprintf(msgAlarmSecond, ord, kind.name, other))
	}
	return pt, nil
}

// sameKindExt is the name of a child of the point whose offnormalAlgorithm is of the kind's
// type, "" when none: that is how a palette extension's kind shows on a station.
func sameKindExt(nodes *pyjson.Object, kind alarmKind) string {
	for _, key := range nodes.Keys() {
		child, ok := strings.CutSuffix(key, "/offnormalAlgorithm")
		if ok && !strings.Contains(child, "/") && get(box.Node(nodes, key), "t") == kind.offnormal {
			return child
		}
	}
	return ""
}

// alarmOps are the ops of one extension under the point with handle h: the add with the plain
// slots, then set ops in the same batch: the offnormal and fault algorithms by type (empty
// value, as n4_set_slot swaps a baja:Simple by simple_type: bench retro 2026-11-15 row 10),
// then every other slot by path.
func alarmOps(h any, pt alarmPoint, inAdd map[string]bool) []any {
	slots, sets := []any{}, []any{}
	set := func(path string, body *pyjson.Object) {
		sets = append(sets, obj("nm", "s", "h", h, "n", pt.name+"/"+path, "b", body))
	}
	set("offnormalAlgorithm", obj("nm", "p", "t", pt.kind.offnormal, "v", ""))
	if pt.kind.fault != "" {
		set("faultAlgorithm", obj("nm", "p", "t", pt.kind.fault, "v", ""))
	}
	for _, w := range pt.writes {
		if inAdd[w.path] {
			slots = append(slots, obj("nm", "p", "n", w.path, "t", w.vt, "v", w.sent))
			continue
		}
		set(w.path, alarmBody(w.vt, w.sent))
	}
	add := obj("nm", "a", "h", h, "n", pt.name, "b", obj("nm", "p", "t", alarmExtType, "s", slots))
	return append([]any{add}, sets...)
}
