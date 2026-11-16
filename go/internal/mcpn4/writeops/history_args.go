package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// historyArgs validates everything that needs no station: the points, the extension name and
// type when given, the pattern, the mode, the interval, the capacity, the change tolerance, the
// full policy and enabled.
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
	var err error
	if name, _ := args.Get("ext_name"); name != nil { // else the type's short name, per point
		if s.name, err = plan.Name("ext_name", name); err != nil {
			return s, nil, err
		}
	}
	if t, ok := args.Get("ext_type"); ok { // one of historyExtTypes (the schema's enum), else the type that fits each point
		s.typ, _ = t.(string)
	}
	if s.mode, err = historyMode(args, s.typ); err != nil {
		return s, nil, err
	}
	pattern, _ := args.Get("history_name")
	if s.pattern, _ = pattern.(string); s.pattern == "" {
		return s, nil, plan.ToolError("history_name must be a non-empty baja:Format pattern, e.g. " +
			"S%parent.parent.parent.name%_%parent.name%")
	}
	if s.interval, err = historyInterval(args, s.mode); err != nil {
		return s, nil, err
	}
	if s.capacity, s.records, err = historyCapacity(args); err != nil {
		return s, nil, err
	}
	if s.tolerance, err = historyTolerance(args, s.mode, s.typ); err != nil {
		return s, nil, err
	}
	s.fullPolicy = historyFullPolicy(args)
	s.enabled = true
	if v, ok := args.Get("enabled"); ok {
		s.enabled = pyjson.Truthy(v)
	}
	return s, points, nil
}
