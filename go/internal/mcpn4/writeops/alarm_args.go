package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// alarmSpec is what the arguments ask for, before any point is read: the points, the kind (""
// for the default of each point's family), the extension name ("" for the kind's palette name),
// the class, the extension slots, the algorithm arguments given (in alarmAlgorithmParams order)
// and their slot writes, and allow_second.
type alarmSpec struct {
	points            []string
	kind, name, class string
	ext               []alarmWrite
	extInAdd          map[string]bool
	params            []string
	paramWrites       map[string][]alarmWrite
	allowSecond       bool
}

// defaultAlarmClass is BAlarmSourceExt.alarmClass's default.
const defaultAlarmClass = "defaultAlarmClass"

// alarmArgs validates everything that needs no station.
func alarmArgs(args *pyjson.Object) (alarmSpec, error) {
	s := alarmSpec{extInAdd: map[string]bool{}, paramWrites: map[string][]alarmWrite{}}
	var err error
	if s.points, err = alarmPoints(args); err != nil {
		return s, err
	}
	if raw, given := args.Get("kind"); given {
		s.kind, _ = raw.(string)
		if _, ok := findAlarmKind(s.kind); !ok {
			return s, plan.ToolError(fmt.Sprintf(msgAlarmKind, strings.Join(alarmKindNames(), ", "), pyjson.Repr(raw)))
		}
	}
	if name, _ := args.Get("ext_name"); name != nil {
		if s.name, err = plan.Name("ext_name", name); err != nil {
			return s, err
		}
	}
	if _, given := args.Get("alarm_class"); !given {
		args = withArg(args, "alarm_class", defaultAlarmClass)
	}
	for _, a := range alarmArgSlots {
		raw, given := args.Get(a.arg)
		if !given {
			continue
		}
		sent, want, err := alarmValue(a.arg, a.vt, raw)
		if err != nil {
			return s, err
		}
		s.ext = append(s.ext, alarmWrite{a.slot, a.vt, sent, want})
		s.extInAdd[a.slot] = a.inAdd
		if a.slot == "alarmClass" {
			s.class = sent.(string)
		}
	}
	for _, p := range alarmAlgorithmParams {
		raw, given := args.Get(p)
		if !given {
			continue
		}
		s.params = append(s.params, p)
		for _, slot := range alarmParamSlots[p] {
			var sent, want any = "true", true // a limit given enables it
			if !strings.HasPrefix(slot.slot, "limitEnable/") {
				if sent, want, err = alarmValue(p, slot.vt, raw); err != nil {
					return s, err
				}
			}
			s.paramWrites[p] = append(s.paramWrites[p], alarmWrite{"offnormalAlgorithm/" + slot.slot, slot.vt, sent, want})
		}
	}
	if v, ok := args.Get("allow_second"); ok {
		s.allowSecond = pyjson.Truthy(v)
	}
	return s, nil
}

func alarmPoints(args *pyjson.Object) ([]string, error) {
	raw, _ := args.Get("points")
	list, _ := raw.([]any)
	if len(list) == 0 || len(list) > maxHistoryPoints {
		return nil, plan.ToolError(fmt.Sprintf(msgAlarmNoPoints, maxHistoryPoints))
	}
	points, dup := []string{}, map[string]bool{}
	for _, p := range list {
		ord, ok := p.(string)
		if !ok || ord == "" || dup[ord] {
			return nil, plan.ToolError(fmt.Sprintf(msgAlarmPointOrd, pyjson.Repr(p)))
		}
		dup[ord] = true
		points = append(points, ord)
	}
	return points, nil
}

// withArg is a copy of args with key set to v.
func withArg(args *pyjson.Object, key string, v any) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range args.Keys() {
		x, _ := args.Get(k)
		out.Set(k, x)
	}
	out.Set(key, v)
	return out
}

// kindTaking names the kinds that take the algorithm argument param.
func kindTaking(param string) string {
	out := []string{}
	for _, k := range alarmKinds {
		if k.takes(param) {
			out = append(out, k.name)
		}
	}
	return strings.Join(out, ", ")
}
