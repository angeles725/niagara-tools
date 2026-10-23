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

// Validate is server._validate: required keys in schema order, then each argument in its
// insertion order against its property's type, "minimum" and "maximum". A bool is never an
// integer or a number; an integer is a number; untyped and unknown properties accept anything.
func Validate(schema, args *pyjson.Object) error {
	props, _ := lookup(schema, "properties").(*pyjson.Object)
	if req, ok := lookup(schema, "required").([]any); ok {
		for _, key := range req {
			k, _ := key.(string)
			if _, ok := args.Get(k); !ok {
				return &InvalidParams{"missing required argument: " + pyjson.Str(key)}
			}
		}
	}
	for _, key := range args.Keys() {
		value, _ := args.Get(key)
		spec, ok := lookup(props, key).(*pyjson.Object)
		if !ok {
			continue
		}
		kind, typed := spec.Get("type")
		if !typed || kind == nil {
			continue
		}
		name, _ := kind.(string)
		if !hasType(value, name) {
			if !known(name) {
				return &SchemaError{fmt.Sprintf("schema for %s uses unsupported type %s", key, pyjson.Repr(kind))}
			}
			return &InvalidParams{fmt.Sprintf("argument %s must be %s", key, name)}
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
				return &SchemaError{fmt.Sprintf("schema for %s: %v", key, err)}
			}
			if comparable && c == bound.bad {
				return &InvalidParams{fmt.Sprintf("argument %s must be %s %s", key, bound.op, pyjson.Str(limit))}
			}
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
