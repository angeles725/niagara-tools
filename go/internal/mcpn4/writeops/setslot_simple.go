package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// simpleWireType is the type a baja:Simple value is sent as: simple_type when given, else the
// slot's own type. A struct (a slot with child slots, such as a Modbus FlexAddress) and a Status
// type are refused: their children are written one by one, a Status whole by its own value type.
func simpleWireType(nodes *pyjson.Object, slot, slotType, simpleType string) (string, error) {
	for _, k := range nodes.Keys() {
		if strings.HasPrefix(k, slot+"/") {
			return "", plan.ToolError("slot " + pyjson.StrRepr(slot) + " is a struct (" + slotType +
				"); set its child slots, e.g. " + k)
		}
	}
	wire := slotType
	if simpleType != "" {
		wire = simpleType
	}
	if plan.IsStatusType(wire) || plan.IsStatusType(slotType) {
		return "", plan.ToolError("baja:Simple does not write Status types; use baja:StatusNumeric or baja:StatusBoolean")
	}
	return wire, nil
}

// simpleBSON is the BOX body of an encoded simple of type t.
func simpleBSON(t string, encoded any) *pyjson.Object {
	o := pyjson.NewObject()
	o.Set("nm", "p")
	o.Set("t", t)
	o.Set("v", encoded)
	return o
}
