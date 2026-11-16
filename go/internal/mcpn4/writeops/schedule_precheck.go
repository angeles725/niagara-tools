package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// unreadSchedule is the refusal when the schedule at ord cannot be read: a missing component is
// named as such, and the faults of the request that need no station (day names, times, overlaps,
// shapes) are listed with it, so one call reports them all.
func unreadSchedule(ord string, args *pyjson.Object, err error) error {
	var f schedule.Faults
	if raw, given := args.Get("weekly"); given {
		if arg, ok := raw.(*pyjson.Object); ok {
			schedule.Weekly(arg, nil, "", &f)
		} else {
			f = append(f, "weekly: must be an object {day: [{start, end, value}]}")
		}
	}
	why := err.Error()
	if strings.Contains(why, "KeyError") {
		why = "no component at " + ord + " (" + why + ")"
	}
	f = append(f, "ord: "+why)
	return plan.ToolError(f.Err().Error())
}
