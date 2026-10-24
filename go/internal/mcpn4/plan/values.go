package plan

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// ValueTypes is VALUE_TYPES: the value types n4_set_slot writes.
var ValueTypes = []string{"baja:Double", "baja:Boolean", "baja:String", "baja:StatusNumeric", "baja:StatusBoolean"}

// statusKind maps a Status type to the type of its value child.
var statusKind = map[string]string{"baja:StatusNumeric": "baja:Double", "baja:StatusBoolean": "baja:Boolean"}

// namePattern and statusPattern are _NAME and _STATUS under re.match: Python's "$" also matches
// before one final "\n", and "." never matches "\n".
var (
	namePattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*\n?$`)
	statusPattern = regexp.MustCompile(`^[0-9a-fA-F]+(;[^\n]*)?\n?$`)
)

// ToolError is a tools_read.ToolError: the kit's own refusal, shown as is.
func ToolError(msg string) error { return &jsonrpc.ToolError{Msg: msg} }

// IsStatusType is value_type.startswith("baja:Status").
func IsStatusType(valueType string) bool { return strings.HasPrefix(valueType, "baja:Status") }

// Coerce is _coerce: value checked for valueType and normalized (a float, a bool, a str, or a
// {value, status} dict for a Status type). An int past the float range is the OverflowError
// math.isfinite raises.
func Coerce(valueType string, value, status any) (any, error) {
	kind := valueType
	if k, ok := statusKind[valueType]; ok {
		kind = k
	}
	var norm any
	switch kind {
	case "baja:Double":
		n, ok := value.(pyjson.Number) // a JSON bool is never a Number
		if !ok {
			return nil, ToolError("value must be a finite number for " + valueType)
		}
		f, err := n.Float64()
		if err != nil {
			return nil, err
		}
		if !finite(f) {
			return nil, ToolError("value must be a finite number for " + valueType)
		}
		norm = f
	case "baja:Boolean":
		if _, ok := value.(bool); !ok {
			return nil, ToolError("value must be true or false for " + valueType)
		}
		norm = value
	case "baja:String":
		if _, ok := value.(string); !ok {
			return nil, ToolError("value must be a string for baja:String")
		}
		norm = value
	default:
		return nil, valueTypeError()
	}
	if !IsStatusType(valueType) {
		return norm, nil
	}
	if s, ok := status.(string); !ok || !statusPattern.MatchString(s) {
		return nil, ToolError("status must be a hex status string such as '0'")
	}
	return obj("value", norm, "status", status), nil
}

func finite(f float64) bool { return f-f == 0 }

func valueTypeError() error {
	return ToolError(safety.ReasonValueType + " " + strings.Join(ValueTypes, ", "))
}

// ToBSON is _to_bson: the BOX body of a normalized value.
func ToBSON(valueType string, norm any) *pyjson.Object {
	switch valueType {
	case "baja:StatusNumeric", "baja:StatusBoolean":
		n := norm.(*pyjson.Object)
		v, _ := n.Get("value")
		s, _ := n.Get("status")
		inner := bsonBool(v)
		if valueType == "baja:StatusNumeric" {
			inner = bsonDouble(v.(float64))
		}
		inner.Set("n", "value")
		return obj("nm", "p", "t", valueType, "s", []any{inner,
			obj("nm", "p", "n", "status", "t", "baja:Status", "v", pyjson.Str(s))})
	case "baja:Double":
		return bsonDouble(norm.(float64))
	case "baja:Boolean":
		return bsonBool(norm)
	}
	return obj("nm", "p", "t", "baja:String", "v", norm)
}

func bsonDouble(f float64) *pyjson.Object {
	return obj("nm", "p", "t", "baja:Double", "v", pyjson.FloatRepr(f))
}

func bsonBool(b any) *pyjson.Object {
	v := "false"
	if pyjson.Truthy(b) {
		v = "true"
	}
	return obj("nm", "p", "t", "baja:Boolean", "v", v)
}

func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
