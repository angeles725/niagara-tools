package tags

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Marker is a marker tag's type and its only value, as a station encodes it.
const (
	Marker      = "baja:Marker"
	MarkerValue = "M"
)

// InferType is the type of an ad hoc tag given no type: a Marker without a value, else the JSON
// value's own kind.
func InferType(value any) string {
	switch value.(type) {
	case nil:
		return Marker
	case bool:
		return "baja:Boolean"
	case pyjson.Number:
		return "baja:Double"
	}
	return "baja:String"
}

// Body is the BOX body of a tag of type typ: value when given, else def (the definition's
// encoded default). A Marker takes no value (or true); Double, Boolean, String and Integer are
// checked as n4_set_slot checks them; any other simple type is sent as its encoded string.
func Body(typ string, value, def any) (*pyjson.Object, error) {
	body := pyjson.NewObject()
	body.Set("nm", "p")
	body.Set("t", typ)
	if typ == Marker {
		if value != nil && value != true {
			return nil, plan.ToolError("a baja:Marker tag has no value: omit value (or pass true)")
		}
		body.Set("v", MarkerValue)
		return body, nil
	}
	if value == nil {
		if def == nil {
			return nil, plan.ToolError("value is required for a " + typ + " tag: its definition sets no default")
		}
		body.Set("v", def)
		return body, nil
	}
	switch typ {
	case "baja:Double", "baja:Boolean", "baja:String", "baja:Integer":
		norm, err := plan.Coerce(typ, value, nil)
		if err != nil {
			return nil, err
		}
		return plan.ToBSON(typ, norm), nil
	}
	s, ok := value.(string)
	if !ok {
		return nil, plan.ToolError("value must be the encoded string of a " + typ + " (as n4_read_tags shows it)")
	}
	body.Set("v", s)
	return body, nil
}
