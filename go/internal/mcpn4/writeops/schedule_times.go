package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// timesPart replaces a trigger's firing times with one set of its times day.
func timesPart(s *scheduleState, args *pyjson.Object, f *schedule.Faults) []schedulePart {
	raw, given := args.Get("times")
	if !given {
		return nil
	}
	if !s.kind.Times {
		*f = append(*f, "times: a "+s.kind.Name+" schedule has none (only a trigger fires at times)")
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		*f = append(*f, "times: must be a list of HH:MM")
		return nil
	}
	times := schedule.TriggerTimes(list, f)
	old, err := previousSlot(s, "times")
	if err != nil {
		*f = append(*f, "times: "+err.Error())
		return nil
	}
	want := []any{}
	for _, m := range times {
		want = append(want, fmt.Sprintf("%02d:%02d", m/60, m%60))
	}
	return []schedulePart{{op: setOp(s.h, "times", schedule.TimesBody(times)), inverse: setOp(s.h, "times", old),
		key: "times", want: want, note: fmt.Sprintf("replaces the times with %d time(s)", len(times))}}
}
