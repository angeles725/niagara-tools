package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// copyRuntimeSlots are slots the driver or the control engine computes; a copy starts without
// them so it never shows the source's state (a down status, a failed ping) as its own.
// absoluteAddress is a Modbus proxy's read-only transient address (flags 3), derived by the driver
// from dataAddress (modbusCore BModbusClientProxyExt). An alarm extension's state is runtime too
// (alarm BAlarmSourceExt): timeInCurrentState ticks, so copying it changed the plan between the
// dry run and the execution; alarmState, ackedTransitions and the transition timestamps are the
// source's alarm history.
var copyRuntimeSlots = map[string]bool{"status": true, "health": true, "faultCause": true,
	"readValue": true, "writeValue": true, "absoluteAddress": true,
	"timeInCurrentState": true, "alarmState": true, "ackedTransitions": true,
	"toOffnormalTimes": true, "toFaultTimes": true}

// stripRuntime is body without copyRuntimeSlots, at every level, and without children the load
// lists by name only (nameOnly) (actions such as upload, frozen slots at their default such as regType):
// an add op child without a type is not a value, and the copy's type supplies those slots.
func stripRuntime(body *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k != "s" {
			out.Set(k, v)
			continue
		}
		kept := []any{}
		for _, c := range v.([]any) {
			kid := c.(*pyjson.Object)
			if n, _ := get(kid, "n").(string); !copyRuntimeSlots[n] && !nameOnly(kid) {
				kept = append(kept, stripRuntime(kid))
			}
		}
		if len(kept) > 0 {
			out.Set("s", kept)
		}
	}
	return out
}

// nameOnly is a loaded child with no type, value or children: an action or a frozen slot at its
// default, listed by name only.
func nameOnly(kid *pyjson.Object) bool {
	return get(kid, "t") == nil && get(kid, "v") == nil && get(kid, "s") == nil
}
