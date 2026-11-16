package writeops

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// alarmWrite is one slot n4_add_alarm writes: its path under the extension, its value type, the
// encoded value the op sends and the value the read-back expects (plan.Observe's form: a float
// for baja:Double, a bool for baja:Boolean, {value, status} for baja:StatusBoolean, else the
// encoded string).
type alarmWrite struct {
	path, vt string
	sent     any
	want     any
}

// alarmArgSlots are the extension slots an argument sets, in op order. inAdd slots go in the add
// body (plain baja values); the others are set ops after it (alarmEnable is an alarm-module
// simple, alarmInhibit a baja:StatusBoolean struct, both written the way n4_set_slot writes
// them).
var alarmArgSlots = []struct {
	arg, slot, vt string
	inAdd         bool
}{
	{"alarm_class", "alarmClass", "baja:String", true}, {"source_name", "sourceName", "baja:Format", true},
	{"to_offnormal_text", "toOffnormalText", "baja:Format", true}, {"to_fault_text", "toFaultText", "baja:Format", true},
	{"to_normal_text", "toNormalText", "baja:Format", true}, {"time_delay_ms", "timeDelay", "baja:RelTime", true},
	{"time_delay_to_normal_ms", "timeDelayToNormal", "baja:RelTime", true},
	{"hyperlink_ord", "hyperlinkOrd", "baja:Ord", true},
	{"alarm_enable", "alarmEnable", "alarm:AlarmTransitionBits", false}, {"inhibit", "alarmInhibit", "baja:StatusBoolean", false},
}

// alarmParamSlots are the algorithm slots (under offnormalAlgorithm) each algorithm argument
// sets: BOutOfRangeAlgorithm highLimit, lowLimit, deadband (double), highLimitText, lowLimitText
// (baja:Format), and the limit's enable bit in limitEnable (alarm:LimitEnable, a struct whose
// lowLimitEnable and highLimitEnable are false by default); BBooleanChangeOfStateAlgorithm
// alarmValue (boolean); B{Enum,Numeric}ChangeOfStateAlgorithm alarmValues (baja:EnumRange).
var alarmParamSlots = map[string][]struct{ slot, vt string }{
	"high_limit":      {{"highLimit", "baja:Double"}, {"limitEnable/highLimitEnable", "baja:Boolean"}},
	"low_limit":       {{"lowLimit", "baja:Double"}, {"limitEnable/lowLimitEnable", "baja:Boolean"}},
	"deadband":        {{"deadband", "baja:Double"}},
	"high_limit_text": {{"highLimitText", "baja:Format"}}, "low_limit_text": {{"lowLimitText", "baja:Format"}},
	"alarm_value": {{"alarmValue", "baja:Boolean"}}, "alarm_values": {{"alarmValues", "baja:EnumRange"}},
}

// alarmValue coerces the argument arg to the value type vt: the encoded value sent and the value
// the read-back expects.
func alarmValue(arg, vt string, raw any) (sent, want any, err error) {
	switch vt {
	case "baja:Double", "baja:Boolean", "baja:Format", "baja:RelTime":
		norm, err := plan.Coerce(vt, raw, nil)
		if err != nil {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmArg, arg, err.Error()))
		}
		if f, ok := norm.(float64); ok {
			if arg == "deadband" && f < 0 {
				return nil, nil, plan.ToolError(msgAlarmDeadband)
			}
			return pyjson.FloatRepr(f), f, nil
		}
		if b, ok := norm.(bool); ok {
			return strconv.FormatBool(b), b, nil
		}
		return norm, norm, nil
	case "baja:StatusBoolean":
		b, ok := raw.(bool)
		if !ok {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmArg, arg, "value must be true or false"))
		}
		return b, obj("value", b, "status", "0"), nil
	case "alarm:AlarmTransitionBits":
		return alarmEnableMask(raw)
	case "baja:EnumRange":
		return alarmEnumRange(raw)
	}
	s, ok := raw.(string) // baja:String, baja:Ord
	if !ok || s == "" {
		return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmString, arg, pyjson.Repr(raw)))
	}
	return s, s, nil
}

// alarmEnableMask is alarm_enable as BAlarmTransitionBits encodes it: the hex bit mask.
func alarmEnableMask(raw any) (sent, want any, err error) {
	list, ok := raw.([]any)
	if !ok {
		return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmEnable, pyjson.Repr(raw)))
	}
	mask := 0
	for _, v := range list {
		s, _ := v.(string)
		bit, known := alarmTransitionBits[s]
		if !known {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmEnable, pyjson.Repr(raw)))
		}
		mask |= bit
	}
	hex := strconv.FormatInt(int64(mask), 16)
	return hex, hex, nil
}

// alarmEnumRange is alarm_values as BEnumRange.encodeToString writes a dynamic range:
// {tag=ordinal,tag=ordinal}, in the order given; each tag must be a plain slot name.
func alarmEnumRange(raw any) (sent, want any, err error) {
	o, ok := raw.(*pyjson.Object)
	if !ok || len(o.Keys()) == 0 {
		return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmValues, pyjson.Repr(raw)))
	}
	out := "{"
	for i, tag := range o.Keys() {
		v, _ := o.Get(tag)
		n, isNum := v.(pyjson.Number)
		ord, isInt := n.BigInt()
		if _, err := plan.Name("alarm_values tag", tag); err != nil || !isNum || !isInt {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgAlarmValues, pyjson.Repr(raw)))
		}
		if i > 0 {
			out += ","
		}
		out += tag + "=" + ord.String()
	}
	out += "}"
	return out, out, nil
}

// alarmBody is the BOX body of an encoded value of type vt: plan.ToBSON for the types it knows,
// else a plain simple of type vt (as n4_set_slot sends a baja:Simple).
func alarmBody(vt string, sent any) *pyjson.Object {
	switch vt {
	case "baja:Double":
		f, _ := plan.Float(sent)
		return plan.ToBSON(vt, f)
	case "baja:Boolean":
		return plan.ToBSON(vt, sent == "true")
	case "baja:StatusBoolean":
		return plan.ToBSON(vt, obj("value", sent, "status", "0"))
	}
	return obj("nm", "p", "t", vt, "v", sent)
}
