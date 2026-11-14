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
func coerceSimple(valueType string, value any) (any, error) {
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
	if valueType == "baja:Format" || valueType == "baja:RelTime" {
		return valueType
	}
	return "baja:String"
}
