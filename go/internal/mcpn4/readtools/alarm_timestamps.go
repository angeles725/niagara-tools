package readtools

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// alarmTimestampsType is the struct an alarm extension keeps per transition (toOffnormalTimes,
// toFaultTimes): javax.baja.alarm.ext.BAlarmTimestamps (alarm-rt), a BStruct of alarmTime,
// ackTime and normalTime (BAbsTime, default BAbsTime.NULL: epoch 0) and count (int, default 0).
const alarmTimestampsType = "alarm:AlarmTimestamps"

// alarmTimestampFields maps each BAlarmTimestamps property to its decoded key.
var alarmTimestampFields = [][2]string{{"alarmTime", "alarm_time"}, {"ackTime", "ack_time"},
	{"normalTime", "normal_time"}, {"count", "count"}}

// alarmTimestamps decodes the slot entries of an alarm:AlarmTimestamps: each time as the
// station's ISO text with its offset, null when unset (epoch 0) or absent, and count as an
// integer (its raw value when not one, 0 when absent).
func alarmTimestamps(slots []any) *pyjson.Object {
	values := map[string]any{}
	for _, s := range slots {
		if e, ok := s.(*pyjson.Object); ok {
			name, _ := e.Get("name")
			if n, ok := name.(string); ok {
				values[n], _ = e.Get("value")
			}
		}
	}
	out := pyjson.NewObject()
	for _, f := range alarmTimestampFields {
		v, ok := values[f[0]]
		switch {
		case f[0] != "count" && (!ok || v == nil):
			out.Set(f[1], nil)
		case f[0] != "count":
			out.Set(f[1], bql.IsoOrNull(pyjson.Str(v)))
		case !ok || v == nil:
			out.Set(f[1], 0)
		default:
			if n, err := strconv.Atoi(pyjson.Str(v)); err == nil {
				out.Set(f[1], n)
			} else {
				out.Set(f[1], v)
			}
		}
	}
	return out
}
