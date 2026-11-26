package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// applyValue is one config a row sets: the slot, the type it is written and read back as, the
// value normalized (what the read-back compares) and the op body.
type applyValue struct {
	slot, valueType string
	norm            any
	body            *pyjson.Object
}

// configValueType is the value type a config slot of the golden is written as, from the slot's own
// type: a plain simple keeps its type, any other simple is sent as its encoded string (wire is its
// type). A Status or a struct is refused: n4_set_slot writes those.
func (g *applyGolden) configValueType(slot string) (valueType, wire string, err error) {
	n := kidNamed(g.cfg, slot)
	t, _ := get(n, "t").(string)
	switch {
	case t == "":
		return "", "", plan.ToolError("config slot " + pyjson.StrRepr(slot) + " has no type in the golden's templateConfig load")
	case plan.IsStatusType(t) || get(n, "s") != nil:
		return "", "", plan.ToolError("config slot " + pyjson.StrRepr(slot) + " is " + t + ", a Status or a struct: " +
			"n4_apply sets plain slots; set it on the copies with n4_set_slot")
	}
	switch t {
	case "baja:Double", "baja:Boolean", "baja:String", "baja:Format", "baja:RelTime", "baja:Integer":
		return t, "", nil
	}
	return "baja:Simple", t, nil
}

// values are the sets of row, in the order its configs were given, each typed from the golden's
// config slot; a config naming no config slot, or a value that does not fit, is refused.
func (g *applyGolden) values(row applyRow) ([]*applyValue, error) {
	out := []*applyValue{}
	for _, slot := range row.configs.Keys() {
		if g.bySlot[slot] == nil {
			return nil, plan.ToolError("row " + pyjson.StrRepr(row.name) + ": config " + pyjson.StrRepr(slot) +
				" names no config slot of the golden's templateConfig (a slot a ConfigBinding names as sourceSlot: " +
				joinNames(g.slotNames()) + ")")
		}
		vt, wire, err := g.configValueType(slot)
		if err != nil {
			return nil, err
		}
		raw, _ := row.configs.Get(slot)
		norm, err := plan.Coerce(vt, raw, nil)
		if err != nil {
			return nil, plan.ToolError("row " + pyjson.StrRepr(row.name) + ", config " + pyjson.StrRepr(slot) + ": " + err.Error())
		}
		body := plan.ToBSON(vt, norm)
		if vt == "baja:Simple" {
			body = simpleBSON(wire, norm)
		}
		out = append(out, &applyValue{slot, vt, norm, body})
	}
	return out, nil
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
