package jsonrpc

import (
	"fmt"
	"math"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// InvalidParams is server.InvalidParams: a -32602 reply carrying Msg.
type InvalidParams struct{ Msg string }

func (e *InvalidParams) Error() string { return e.Msg }

// SchemaError is the ValueError _validate raises for a schema it cannot check: a server bug,
// never a client error.
type SchemaError struct{ Msg string }

func (e *SchemaError) Error() string { return e.Msg }

// KnownTypes are the JSON schema "type" values Validate checks (server._JSON_TYPES).
var KnownTypes = []string{"string", "boolean", "integer", "number", "array", "object"}

// Validate checks tool's arguments against its input schema before anything reaches a station:
// required keys in schema order, then each argument in its insertion order against its
// property's type, "minimum", "maximum" and "enum". An argument the schema does not declare is
// refused (retro row M11: a dropped argument is a silent wrong write), unless the schema keeps
// its object open with "additionalProperties". Nested objects are checked the same way only
// where their schema declares "properties"; array items only for "enum" and declared
// "properties". A bool is never an integer or a number; an integer is a number; an untyped
// property (anyOf) accepts any value and is left to its tool.
func Validate(tool string, schema, args *pyjson.Object) error {
	return validateObject(tool, "", schema, args)
}

// checkValue is one argument's type, bounds and enum; name is its path (points[0].ord).
func checkValue(tool, name string, spec *pyjson.Object, value any) error {
	if err := checkTyped(name, spec, value); err != nil {
		return err
	}
	if err := checkEnum(name, spec, value); err != nil {
		return err
	}
	return checkNested(tool, name, spec, value)
}

// checkTyped is the type and the "minimum"/"maximum" bounds of a typed property.
func checkTyped(name string, spec *pyjson.Object, value any) error {
	kind, typed := spec.Get("type")
	if !typed || kind == nil {
		return nil
	}
	tname, _ := kind.(string)
	if !hasType(value, tname) {
		if !known(tname) {
			return &SchemaError{fmt.Sprintf("schema for %s uses unsupported type %s", name, pyjson.Repr(kind))}
		}
		return &InvalidParams{fmt.Sprintf("argument %s must be %s", name, tname)}
	}
	for _, bound := range []struct {
		key, op string
		bad     int // the comparison result that refuses the value
	}{{"minimum", ">=", -1}, {"maximum", "<=", 1}} {
		limit, ok := spec.Get(bound.key)
		if !ok {
			continue
		}
		c, comparable, err := compare(value, limit)
		if err != nil {
			return &SchemaError{fmt.Sprintf("schema for %s: %v", name, err)}
		}
		if comparable && c == bound.bad {
			return &InvalidParams{fmt.Sprintf("argument %s must be %s %s", name, bound.op, pyjson.Str(limit))}
		}
	}
	return nil
}

func lookup(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	v, _ := o.Get(key)
	return v
}

func known(name string) bool {
	for _, k := range KnownTypes {
		if k == name {
			return true
		}
	}
	return false
}

// hasType is isinstance(value, _JSON_TYPES[name]) minus bools for the numeric kinds; an
// unknown name matches nothing.
func hasType(v any, name string) bool {
	switch x := v.(type) {
	case string:
		return name == "string"
	case bool:
		return name == "boolean"
	case pyjson.Number:
		return name == "number" || name == "integer" && !x.IsFloat()
	case []any:
		return name == "array"
	case *pyjson.Object:
		return name == "object"
	}
	return false
}

// compare orders two numbers exactly, as Python compares int and float (no rounding of a big
// int). comparable is false when a NaN is involved: every comparison with it is False.
func compare(a, b any) (c int, comparable bool, err error) {
	x, okA := a.(pyjson.Number)
	y, okB := b.(pyjson.Number)
	if !okA || !okB {
		return 0, false, fmt.Errorf("cannot compare %s with %s", pyjson.Repr(a), pyjson.Repr(b))
	}
	if xi, ok := x.BigInt(); ok {
		if yi, ok := y.BigInt(); ok {
			return xi.Cmp(yi), true, nil
		}
	}
	fx, okx := exact(x)
	fy, oky := exact(y)
	if !okx || !oky {
		return 0, false, nil
	}
	return fx.Cmp(fy), true, nil
}

// exact is n as an exact big.Float (±Inf included); false for NaN.
func exact(n pyjson.Number) (*big.Float, bool) {
	if i, ok := n.BigInt(); ok {
		return new(big.Float).SetInt(i), true
	}
	f, _ := n.Float64()
	if math.IsNaN(f) {
		return nil, false
	}
	return new(big.Float).SetFloat64(f), true
}
