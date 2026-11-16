package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// ptStr is a string point; ptA, ptBool and ptEnum are the history tests' points.
const ptStr = "station:|slot:/Folder/Text"

// alarmClasses is the station's answer to the alarm class query: two classes.
const alarmClasses = "Slot Path\nslot:/Services/AlarmService/defaultAlarmClass\nslot:/Services/AlarmService/Critical\n"

// alarmPoints serves a numeric (ptA), boolean, enum and string point and the class table.
func alarmPoints(t *testing.T, kv ...any) *histClient {
	t.Helper()
	all := append([]any{ptA, point("10"), ptBool, typedPoint("control:BooleanWritable", "11"),
		ptEnum, typedPoint("control:EnumWritable", "12"), ptStr, typedPoint("control:StringPoint", "13")}, kv...)
	c := histPoints(t, all...)
	c.ids = alarmClasses
	return c
}

func alarmArgsOf(points []any, kv ...any) *pyjson.Object {
	a := obj("points", points)
	for i := 0; i < len(kv); i += 2 {
		a.Set(kv[i].(string), kv[i+1])
	}
	return a
}

func planAlarm(c *histClient, args *pyjson.Object) (*plan.Planned, error) {
	return writeops.AddAlarm.PlanWith(c, args, nil, histSession)
}

// opsJSON is every op of p, one per line.
func opsJSON(t *testing.T, p *plan.Planned) string {
	t.Helper()
	lines := []string{}
	for _, op := range p.Ops {
		lines = append(lines, js(t, op))
	}
	return strings.Join(lines, "\n")
}

func TestAddAlarmPicksTheKindFromThePointType(t *testing.T) {
	c := alarmPoints(t)
	p, err := planAlarm(c, alarmArgsOf([]any{ptA, ptBool, ptEnum, ptStr}))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"nm": "a", "h": "10", "n": "OutOfRangeAlarmExt", "b": {"nm": "p", "t": "alarm:AlarmSourceExt", "s": [{"nm": "p", "n": "alarmClass", "t": "baja:String", "v": "defaultAlarmClass"}]}}
{"nm": "s", "h": "10", "n": "OutOfRangeAlarmExt/offnormalAlgorithm", "b": {"nm": "p", "t": "alarm:OutOfRangeAlgorithm", "v": ""}}
{"nm": "s", "h": "10", "n": "OutOfRangeAlarmExt/faultAlgorithm", "b": {"nm": "p", "t": "alarm:OutOfRangeFaultAlgorithm", "v": ""}}
{"nm": "a", "h": "11", "n": "BooleanChangeOfStateAlarmExt", "b": {"nm": "p", "t": "alarm:AlarmSourceExt", "s": [{"nm": "p", "n": "alarmClass", "t": "baja:String", "v": "defaultAlarmClass"}]}}
{"nm": "s", "h": "11", "n": "BooleanChangeOfStateAlarmExt/offnormalAlgorithm", "b": {"nm": "p", "t": "alarm:BooleanChangeOfStateAlgorithm", "v": ""}}
{"nm": "a", "h": "12", "n": "EnumChangeOfStateAlarmExt", "b": {"nm": "p", "t": "alarm:AlarmSourceExt", "s": [{"nm": "p", "n": "alarmClass", "t": "baja:String", "v": "defaultAlarmClass"}]}}
{"nm": "s", "h": "12", "n": "EnumChangeOfStateAlarmExt/offnormalAlgorithm", "b": {"nm": "p", "t": "alarm:EnumChangeOfStateAlgorithm", "v": ""}}
{"nm": "s", "h": "12", "n": "EnumChangeOfStateAlarmExt/faultAlgorithm", "b": {"nm": "p", "t": "alarm:EnumFaultAlgorithm", "v": ""}}
{"nm": "a", "h": "13", "n": "StringChangeOfValueAlarmExt", "b": {"nm": "p", "t": "alarm:AlarmSourceExt", "s": [{"nm": "p", "n": "alarmClass", "t": "baja:String", "v": "defaultAlarmClass"}]}}
{"nm": "s", "h": "13", "n": "StringChangeOfValueAlarmExt/offnormalAlgorithm", "b": {"nm": "p", "t": "alarm:StringChangeOfStateAlgorithm", "v": ""}}
{"nm": "s", "h": "13", "n": "StringChangeOfValueAlarmExt/faultAlgorithm", "b": {"nm": "p", "t": "alarm:StringChangeOfStateFaultAlgorithm", "v": ""}}`
	if got := opsJSON(t, p); got != want {
		t.Fatalf("ops\n got  %s\n want %s", got, want)
	}
	if got := js(t, p.Inverse[1]); got != `{"nm": "v", "h": "11", "n": "BooleanChangeOfStateAlarmExt"}` {
		t.Errorf("inverse[1] = %s", got)
	}
	if len(c.ords) != 1 || !strings.Contains(c.ords[0], "alarm:AlarmClass") {
		t.Errorf("BQL reads = %v, want one class read", c.ords)
	}
	st, err := planAlarm(c, alarmArgsOf([]any{ptBool}, "kind", "status"))
	if err != nil || !strings.Contains(opsJSON(t, st), `"n": "StatusAlarmExt/faultAlgorithm", "b": {"nm": "p", "t": "alarm:StatusFaultAlgorithm"`) {
		t.Errorf("status on a boolean point: err = %v", err)
	}
}

func TestAddAlarmSetsTextsDelaysEnablesAndLimits(t *testing.T) {
	c := alarmPoints(t)
	p, err := planAlarm(c, alarmArgsOf([]any{ptA}, "high_limit", pyjson.IntLiteral("80"), "deadband", pyjson.IntLiteral("1"),
		"high_limit_text", "%alarmData.sourceName% high: %alarmData.presentValue% > %alarmData.highLimit%",
		"to_offnormal_text", "off", "to_fault_text", "fault", "to_normal_text", "back to normal",
		"source_name", "AHU1 %parent.displayName%", "alarm_class", "Critical", "time_delay_ms", pyjson.IntLiteral("60000"),
		"time_delay_to_normal_ms", pyjson.IntLiteral("0"), "alarm_enable", []any{"offnormal", "normal"},
		"inhibit", true, "hyperlink_ord", "station:|slot:/Graphics", "ext_name", "HighTemp"))
	if err != nil {
		t.Fatal(err)
	}
	ops := opsJSON(t, p)
	for _, want := range []string{
		`{"nm": "a", "h": "10", "n": "HighTemp", "b": {"nm": "p", "t": "alarm:AlarmSourceExt", "s": [` +
			`{"nm": "p", "n": "alarmClass", "t": "baja:String", "v": "Critical"}, ` +
			`{"nm": "p", "n": "sourceName", "t": "baja:Format", "v": "AHU1 %parent.displayName%"}, ` +
			`{"nm": "p", "n": "toOffnormalText", "t": "baja:Format", "v": "off"}, ` +
			`{"nm": "p", "n": "toFaultText", "t": "baja:Format", "v": "fault"}, ` +
			`{"nm": "p", "n": "toNormalText", "t": "baja:Format", "v": "back to normal"}, ` +
			`{"nm": "p", "n": "timeDelay", "t": "baja:RelTime", "v": "60000"}, ` +
			`{"nm": "p", "n": "timeDelayToNormal", "t": "baja:RelTime", "v": "0"}, ` +
			`{"nm": "p", "n": "hyperlinkOrd", "t": "baja:Ord", "v": "station:|slot:/Graphics"}]}}`,
		`{"nm": "s", "h": "10", "n": "HighTemp/alarmEnable", "b": {"nm": "p", "t": "alarm:AlarmTransitionBits", "v": "5"}}`,
		`{"nm": "s", "h": "10", "n": "HighTemp/alarmInhibit", "b": {"nm": "p", "t": "baja:StatusBoolean", "s": [`,
		`{"nm": "s", "h": "10", "n": "HighTemp/offnormalAlgorithm/highLimit", "b": {"nm": "p", "t": "baja:Double", "v": "80.0"}}`,
		`{"nm": "s", "h": "10", "n": "HighTemp/offnormalAlgorithm/limitEnable/highLimitEnable", "b": {"nm": "p", "t": "baja:Boolean", "v": "true"}}`,
		`{"nm": "s", "h": "10", "n": "HighTemp/offnormalAlgorithm/deadband", "b": {"nm": "p", "t": "baja:Double", "v": "1.0"}}`,
		`"n": "HighTemp/offnormalAlgorithm/highLimitText", "b": {"nm": "p", "t": "baja:Format", "v": "%alarmData.sourceName% high: %alarmData.presentValue% > %alarmData.highLimit%"}}`,
	} {
		if !strings.Contains(ops, want) {
			t.Errorf("ops lack %s\n%s", want, ops)
		}
	}
	if strings.Contains(ops, "lowLimitEnable") {
		t.Error("a limit not given was enabled")
	}
	for _, tc := range []struct {
		enable []any
		want   string
	}{{[]any{}, `"v": "0"`}, {[]any{"offnormal", "fault", "normal", "alert"}, `"v": "f"`}, {[]any{"fault"}, `"v": "2"`}} {
		p, err := planAlarm(c, alarmArgsOf([]any{ptA}, "alarm_enable", tc.enable))
		if err != nil || !strings.Contains(opsJSON(t, p), `"t": "alarm:AlarmTransitionBits", `+tc.want) {
			t.Errorf("alarm_enable %v: err = %v, want %s", tc.enable, err, tc.want)
		}
	}
	b, err := planAlarm(c, alarmArgsOf([]any{ptBool}, "alarm_value", false))
	if err != nil || !strings.Contains(opsJSON(t, b), `"n": "BooleanChangeOfStateAlarmExt/offnormalAlgorithm/alarmValue", "b": {"nm": "p", "t": "baja:Boolean", "v": "false"}`) {
		t.Errorf("alarm_value: err = %v", err)
	}
	e, err := planAlarm(c, alarmArgsOf([]any{ptEnum}, "alarm_values", obj("alarm", pyjson.IntLiteral("1"), "fault", pyjson.IntLiteral("3"))))
	if err != nil || !strings.Contains(opsJSON(t, e), `"t": "baja:EnumRange", "v": "{alarm=1,fault=3}"`) {
		t.Errorf("alarm_values: err = %v", err)
	}
}

func TestAddAlarmRefusals(t *testing.T) {
	second := obj("", obj("h", "10", "t", "bacnet:BacnetNumericPoint"), "Old", obj("h", "20", "t", "alarm:AlarmSourceExt"),
		"Old/offnormalAlgorithm", obj("t", "alarm:OutOfRangeAlgorithm"))
	for _, tc := range []struct {
		name string
		args *pyjson.Object
		c    *histClient
		want string
	}{
		{"kind mismatch", alarmArgsOf([]any{ptBool}, "kind", "out_of_range"), alarmPoints(t),
			"kind out_of_range does not fit " + ptBool + ", a boolean point (control:BooleanWritable): use one of " +
				"boolean_change_of_state, boolean_command_failure, status, or leave kind out"},
		{"unknown kind", alarmArgsOf([]any{ptA}, "kind", "floating"), alarmPoints(t), "kind must be one of out_of_range, "},
		{"missing class", alarmArgsOf([]any{ptA}, "alarm_class", "Nope"), alarmPoints(t),
			"alarm class 'Nope' does not exist on the station; existing classes: defaultAlarmClass, Critical."},
		{"second extension", alarmArgsOf([]any{ptA}), alarmPoints(t, ptA, second),
			ptA + " already has a out_of_range alarm extension, Old: configure that one with n4_set_slot, or pass allow_second=true"},
		{"name taken", alarmArgsOf([]any{ptA}, "ext_name", "Old", "kind", "numeric_change_of_state"), alarmPoints(t, ptA, second),
			ptA + " already has a child named 'Old'"},
		{"param of another kind", alarmArgsOf([]any{ptA}, "alarm_value", true), alarmPoints(t),
			"alarm_value does not apply to kind out_of_range of " + ptA + ": it is an argument of boolean_change_of_state"},
		{"bad enable", alarmArgsOf([]any{ptA}, "alarm_enable", []any{"high"}), alarmPoints(t),
			"alarm_enable must be a list of offnormal, fault, normal and alert"},
		{"negative deadband", alarmArgsOf([]any{ptA}, "deadband", pyjson.IntLiteral("-1")), alarmPoints(t), "deadband must be 0 or more"},
		{"text limit", alarmArgsOf([]any{ptA}, "high_limit", "80"), alarmPoints(t), "high_limit: value must be a finite number"},
		{"not a point", alarmArgsOf([]any{ptA}), alarmPoints(t, ptA, typedPoint("baja:Folder", "10")),
			ptA + " is a baja:Folder, not a numeric, boolean, enum or string control point"},
		{"bad values", alarmArgsOf([]any{ptEnum}, "alarm_values", obj()), alarmPoints(t), "alarm_values must be a non-empty object"},
	} {
		_, err := planAlarm(tc.c, tc.args)
		wantRefusal(t, tc.name, err, tc.want)
	}
	if _, err := planAlarm(alarmPoints(t, ptA, second), alarmArgsOf([]any{ptA}, "allow_second", true)); err != nil {
		t.Errorf("allow_second: %v", err)
	}
	if _, err := planAlarm(alarmPoints(t, ptA, second), alarmArgsOf([]any{ptA}, "kind", "numeric_change_of_state")); err != nil {
		t.Errorf("another kind beside it: %v", err)
	}
}
