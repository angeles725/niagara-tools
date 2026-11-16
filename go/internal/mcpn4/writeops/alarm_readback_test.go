package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// doneAlarm is the numeric point holding extension name as a station left it after a batch that
// set highLimit to 80, plus extra nodes (path under the extension, node).
func doneAlarm(name, highLimit string, extra ...any) *pyjson.Object {
	p := point("10", name, obj("h", "20", "t", "alarm:AlarmSourceExt"),
		name+"/alarmClass", obj("t", "baja:String", "v", "defaultAlarmClass"),
		name+"/toOffnormalText", obj("t", "baja:Format", "v", "hot"),
		name+"/alarmEnable", obj("t", "alarm:AlarmTransitionBits", "v", "5"),
		name+"/offnormalAlgorithm", obj("h", "21", "t", "alarm:OutOfRangeAlgorithm"),
		name+"/faultAlgorithm", obj("h", "22", "t", "alarm:OutOfRangeFaultAlgorithm"),
		name+"/offnormalAlgorithm/highLimit", obj("t", "baja:Double", "v", highLimit),
		name+"/offnormalAlgorithm/limitEnable", obj("t", "alarm:LimitEnable"),
		name+"/offnormalAlgorithm/limitEnable/highLimitEnable", obj("t", "baja:Boolean", "v", "true"))
	for i := 0; i < len(extra); i += 2 {
		p.Set(name+"/"+extra[i].(string), extra[i+1])
	}
	return p
}

func TestAddAlarmReadbackReportsCoercedValuesAndTheInverseUsesTheAssignedName(t *testing.T) {
	c := alarmPoints(t)
	args := alarmArgsOf([]any{ptA}, "high_limit", pyjson.IntLiteral("80"), "to_offnormal_text", "hot",
		"alarm_enable", []any{"offnormal", "normal"})
	p, err := planAlarm(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "OutOfRangeAlarmExt1")}}
	for range p.Ops[1:] {
		replies = append(replies, []any{})
	}
	inverse, err := writeops.AddAlarm.Inverse(p, replies)
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, inverse); got != `[{"nm": "v", "h": "10", "n": "OutOfRangeAlarmExt1"}]` {
		t.Fatalf("inverse = %s", got)
	}
	if partial, _ := writeops.AddAlarm.Inverse(p, nil); len(partial) != 0 {
		t.Errorf("no add sent: inverse = %s", js(t, partial))
	}
	c.nodes = obj(ptA, doneAlarm("OutOfRangeAlarmExt1", "80.0"))
	requested, _, observed, verdict, err := writeops.AddAlarm.Readback(c, args, p, replies, inverse)
	if err != nil || verdict != "verified" {
		t.Fatalf("verdict = %s, err = %v, observed %s", verdict, err, js(t, observed))
	}
	want := `[{"ord": "` + ptA + `", "name": "OutOfRangeAlarmExt1", "type": "alarm:AlarmSourceExt", ` +
		`"offnormalAlgorithm": "alarm:OutOfRangeAlgorithm", "faultAlgorithm": "alarm:OutOfRangeFaultAlgorithm", ` +
		`"slots": {"alarmClass": "defaultAlarmClass", "toOffnormalText": "hot", "alarmEnable": "5", ` +
		`"offnormalAlgorithm/highLimit": 80.0, "offnormalAlgorithm/limitEnable/highLimitEnable": true}}]`
	if got := js(t, requested); got != want {
		t.Errorf("requested\n got  %s\n want %s", got, want)
	}
	c.nodes = obj(ptA, doneAlarm("OutOfRangeAlarmExt1", "79.5"))
	_, _, observed, verdict, _ = writeops.AddAlarm.Readback(c, args, p, replies, inverse)
	got := js(t, observed.([]any)[0].(*pyjson.Object))
	if verdict != "mismatch" || !strings.Contains(got, `"coerced": {"offnormalAlgorithm/highLimit": {"sent": "80.0", "read": 79.5}}`) {
		t.Errorf("coerced limit: verdict = %s, observed %s", verdict, got)
	}
	swapped := doneAlarm("OutOfRangeAlarmExt1", "80.0", "offnormalAlgorithm", obj("h", "21", "t", "alarm:OffnormalAlgorithm"))
	c.nodes = obj(ptA, swapped)
	if _, _, _, verdict, _ = writeops.AddAlarm.Readback(c, args, p, replies, inverse); verdict != "mismatch" {
		t.Errorf("algorithm not swapped: verdict = %s", verdict)
	}
	c.nodes = obj(ptA, point("10"))
	if _, _, _, verdict, _ = writeops.AddAlarm.Readback(c, args, p, replies, inverse); verdict != "mismatch" {
		t.Errorf("extension missing: verdict = %s", verdict)
	}
}
