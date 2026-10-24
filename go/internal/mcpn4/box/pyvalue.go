package box

import (
	"math"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The helpers below give decoded JSON values (pyjson types) the Python semantics box.py relies
// on: ==, iteration and dict.get (truth tests are pyjson.Truthy).

// numeric returns a number or bool as an exact rational (Python compares int, float and bool
// by value), or false for anything else and for nan/inf.
func numeric(v any) (*big.Rat, bool) {
	switch x := v.(type) {
	case bool:
		if x {
			return big.NewRat(1, 1), true
		}
		return new(big.Rat), true
	case int:
		return new(big.Rat).SetInt64(int64(x)), true
	case float64: // a float the server computed, e.g. float(v) of a read-back
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return nil, false
		}
		return new(big.Rat).SetFloat64(x), true
	case pyjson.Number:
		if i, ok := x.BigInt(); ok {
			return new(big.Rat).SetInt(i), true
		}
		f, _ := x.Float64()
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil, false
		}
		return new(big.Rat).SetFloat64(f), true
	}
	return nil, false
}

// PyEqual is a == b for decoded values (a dict compares by keys, not by order).
func PyEqual(a, b any) bool {
	if ra, ok := numeric(a); ok {
		rb, ok := numeric(b)
		return ok && ra.Cmp(rb) == 0
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i := range x {
			if !PyEqual(x[i], y[i]) {
				return false
			}
		}
		return true
	case *pyjson.Object:
		y, ok := b.(*pyjson.Object)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, k := range x.Keys() {
			xv, _ := x.Get(k)
			yv, ok := y.Get(k)
			if !ok || !PyEqual(xv, yv) {
				return false
			}
		}
		return true
	}
	return false
}

// jsonKind is box._json_kind: a decoded value's JSON kind with its article ("an object", ...).
func jsonKind(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "a boolean"
	case pyjson.Number:
		return "a number"
	case string:
		return "a string"
	case []any:
		return "a list"
	}
	return "an object"
}

// nestsDeeper is box._nests_deeper: whether lists and objects nest deeper than limit in a
// decoded value (the outer one counted). Iterative, like the Python walk.
func nestsDeeper(v any, limit int) bool {
	type item struct {
		v     any
		depth int
	}
	stack := []item{{v, 1}}
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		var kids []any
		switch x := it.v.(type) {
		case []any:
			kids = x
		case *pyjson.Object:
			for _, k := range x.Keys() {
				kid, _ := x.Get(k)
				kids = append(kids, kid)
			}
		default:
			continue
		}
		if it.depth > limit {
			return true
		}
		for _, kid := range kids {
			stack = append(stack, item{kid, it.depth + 1})
		}
	}
	return false
}

// get is v.get(k) when v is a dict: the value and true, else nil and false.
func get(v any, k string) (any, bool) {
	o, ok := v.(*pyjson.Object)
	if !ok {
		return nil, false
	}
	return o.Get(k)
}

// with is dict(o, **{k: v}): a copy of o with k set (in place when present, else appended).
func with(o *pyjson.Object, k string, v any) *pyjson.Object {
	out := pyjson.NewObject()
	for _, key := range o.Keys() {
		val, _ := o.Get(key)
		out.Set(key, val)
	}
	out.Set(k, v)
	return out
}

// obj builds a dict from key, value pairs in order.
func obj(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}
