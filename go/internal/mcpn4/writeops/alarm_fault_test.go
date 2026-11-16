package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A fault limit of out_of_range is set on faultAlgorithm (BOutOfRangeFaultAlgorithm: highLimit,
// lowLimit, deadband, limitEnable) and enables that limit, like the offnormal ones.
func TestAddAlarmSetsAndEnablesFaultLimits(t *testing.T) {
	c := alarmPoints(t)
	p, err := planAlarm(c, alarmArgsOf([]any{ptA}, "fault_high_limit", pyjson.IntLiteral("100"),
		"fault_deadband", pyjson.IntLiteral("2")))
	if err != nil {
		t.Fatal(err)
	}
	ops := opsJSON(t, p)
	for _, want := range []string{
		`{"nm": "s", "h": "10", "n": "OutOfRangeAlarmExt/faultAlgorithm/highLimit", "b": {"nm": "p", "t": "baja:Double", "v": "100.0"}}`,
		`{"nm": "s", "h": "10", "n": "OutOfRangeAlarmExt/faultAlgorithm/limitEnable/highLimitEnable", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}}`,
		`{"nm": "s", "h": "10", "n": "OutOfRangeAlarmExt/faultAlgorithm/deadband", "b": {"nm": "p", "t": "baja:Double", "v": "2.0"}}`,
	} {
		if !strings.Contains(ops, want) {
			t.Errorf("ops lack %s\n%s", want, ops)
		}
	}
	if strings.Contains(ops, "lowLimitEnable") || strings.Contains(ops, "offnormalAlgorithm/highLimit") {
		t.Errorf("a limit not given was set:\n%s", ops)
	}
	if strings.Index(ops, "faultAlgorithm/highLimit") < strings.Index(ops, `"n": "OutOfRangeAlarmExt/faultAlgorithm", `) {
		t.Error("a fault limit is set before the fault algorithm swap")
	}
	low, err := planAlarm(c, alarmArgsOf([]any{ptA}, "fault_low_limit", pyjson.IntLiteral("-5")))
	if err != nil || !strings.Contains(opsJSON(t, low), `"n": "OutOfRangeAlarmExt/faultAlgorithm/limitEnable/lowLimitEnable", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}`) {
		t.Errorf("fault_low_limit: err = %v", err)
	}
	_, err = planAlarm(c, alarmArgsOf([]any{ptBool}, "fault_high_limit", pyjson.IntLiteral("1")))
	wantRefusal(t, "fault limit on a boolean point", err, "fault_high_limit does not apply to kind boolean_change_of_state of "+
		ptBool+": it is an argument of out_of_range")
	_, err = planAlarm(c, alarmArgsOf([]any{ptA}, "fault_deadband", pyjson.IntLiteral("-1")))
	wantRefusal(t, "negative fault deadband", err, "fault_deadband must be 0 or more")
}
