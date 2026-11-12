package plan

import (
	"math"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// coerceSimple checks a baja:Format or baja:RelTime value and normalizes it to the string the
// station encodes it as (BSimple.encodeToString), so it is sent, observed and compared as that
// string: a Format is its pattern ("%parent.name%"); a RelTime is a whole number of
// milliseconds, 0 to 2^63-1 (BRelTime.encodeToString is String.valueOf(ms), 5 minutes is
// 300000).
//
// A baja:Integer is a whole number in the Java int range, sent as its decimal string. A
// baja:Simple is any string (a whole number is taken as its decimal string): the encoded form of the slot's type (an enum's tag, a Facets or a
// LinearConversion string), checked by the station's decodeFromString.
func coerceSimple(valueType string, value any) (any, error) {
	switch valueType {
	case "baja:Simple":
		if n, ok := value.(pyjson.Number); ok { // an MCP client may send "40014" as a number
			if i, isInt := n.BigInt(); isInt {
				return i.String(), nil
			}
		}
		s, ok := value.(string)
		if !ok {
			return nil, ToolError("value must be a string or a whole number (the encoded form, e.g. an enum tag) for baja:Simple")
		}
		return s, nil
	case "baja:Integer":
		n, ok := value.(pyjson.Number)
		i, isInt := n.BigInt()
		if !ok || !isInt || i.Cmp(big.NewInt(math.MinInt32)) < 0 || i.Cmp(big.NewInt(math.MaxInt32)) > 0 {
			return nil, ToolError("value must be a whole number from -2147483648 to 2147483647 for baja:Integer")
		}
		return i.String(), nil
	}
	if valueType == "baja:Format" {
		s, ok := value.(string)
		if !ok {
			return nil, ToolError("value must be a string for baja:Format")
		}
		return s, nil
	}
	n, ok := value.(pyjson.Number) // a JSON bool is never a Number
	ms, isInt := n.BigInt()
	if !ok || !isInt || ms.Sign() < 0 || ms.Cmp(big.NewInt(math.MaxInt64)) > 0 {
		return nil, ToolError("value must be a whole number of milliseconds (0 or more) for baja:RelTime")
	}
	return ms.String(), nil
}

// plainType is the BSON type of a plain (not Status) slot whose value is a string.
func plainType(valueType string) string {
	if valueType == "baja:Format" || valueType == "baja:RelTime" || valueType == "baja:Integer" {
		return valueType
	}
	return "baja:String"
}
