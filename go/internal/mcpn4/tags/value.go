package tags

import (
	"regexp"
	"slices"
	"strings"

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

// ValueTypes are the tag value types n4_set_tags writes: simple values only (a component or a
// struct type is never a tag value).
var ValueTypes = []string{Marker, "baja:Boolean", "baja:Double", "baja:Float", "baja:Integer", "baja:Long",
	"baja:String", "baja:DynamicEnum", "baja:Format", "baja:RelTime"}

// dynamicEnum is BDynamicEnum.encodeToString: the ordinal, then @ and the range when it has one.
var dynamicEnum = regexp.MustCompile(`^-?[0-9]{1,10}(@.+)?$`)

// Body is the BOX body of a tag of type typ: value when given, else def (the definition's
// encoded default). A type outside ValueTypes is refused; a Marker takes no value (or true);
// every other value is checked for its type before it is encoded.
func Body(typ string, value, def any) (*pyjson.Object, error) {
	if !slices.Contains(ValueTypes, typ) {
		return nil, plan.ToolError("type " + pyjson.StrRepr(typ) + " is not a tag value type; one of " +
			strings.Join(ValueTypes, ", "))
	}
	body := pyjson.NewObject()
	body.Set("nm", "p")
	body.Set("t", typ)
	switch {
	case typ == Marker:
		if value != nil && value != true {
			return nil, plan.ToolError("a baja:Marker tag has no value: omit value (or pass true)")
		}
		body.Set("v", MarkerValue)
		return body, nil
	case value == nil && def == nil:
		return nil, plan.ToolError("value is required for a " + typ + " tag: its definition sets no default")
	case value == nil:
		body.Set("v", def)
		return body, nil
	}
	enc, err := encode(typ, value)
	if err != nil {
		return nil, err
	}
	body.Set("v", enc)
	return body, nil
}

// encode is the station's string form of value as a typ, or the refusal naming what typ takes.
func encode(typ string, value any) (any, error) {
	switch typ {
	case "baja:Double", "baja:Float":
		norm, err := plan.Coerce("baja:Double", value, nil)
		if err != nil {
			return nil, plan.ToolError("value must be a finite number for " + typ)
		}
		return pyjson.FloatRepr(norm.(float64)), nil
	case "baja:Long":
		n, ok := value.(pyjson.Number)
		i, isInt := n.BigInt()
		if !ok || !isInt || !i.IsInt64() {
			return nil, plan.ToolError("value must be a whole number (64-bit) for baja:Long")
		}
		return i.String(), nil
	case "baja:DynamicEnum":
		if n, ok := value.(pyjson.Number); ok {
			if i, isInt := n.BigInt(); isInt && i.IsInt64() && dynamicEnum.MatchString(i.String()) {
				return i.String(), nil
			}
		}
		if s, ok := value.(string); ok && dynamicEnum.MatchString(s) {
			return s, nil
		}
		return nil, plan.ToolError("value must be an ordinal or ordinal@range (the encoded form) for baja:DynamicEnum")
	}
	norm, err := plan.Coerce(typ, value, nil) // Boolean, String, Integer, Format, RelTime
	if err != nil {
		return nil, err
	}
	return get(plan.ToBSON(typ, norm), "v"), nil
}
