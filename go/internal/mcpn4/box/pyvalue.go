package box

import (
	"math"
	"math/big"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The helpers below give decoded JSON values (pyjson types) the Python semantics box.py relies
// on: truth tests, ==, iteration and dict.get.

// truthy is bool(v).
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case pyjson.Number:
		if x.IsFloat() {
			f, _ := x.Float64()
			return f != 0
		}
		i, _ := x.BigInt()
		return i.Sign() != 0
	case []any:
		return len(x) > 0
	case *pyjson.Object:
		return x.Len() > 0
	}
	return true
}

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

// pyEqual is a == b for decoded values (a dict compares by keys, not by order).
func pyEqual(a, b any) bool {
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
			if !pyEqual(x[i], y[i]) {
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
			if !ok || !pyEqual(xv, yv) {
				return false
			}
		}
		return true
	}
	return false
}

// pyIter is list(v) for a value a for loop walks: a list's items, a dict's keys, a string's
// characters. Anything else raises TypeError, as iterating None or a number does.
func pyIter(v any) ([]any, error) {
	switch x := v.(type) {
	case []any:
		return x, nil
	case *pyjson.Object:
		out := []any{}
		for _, k := range x.Keys() {
			out = append(out, k)
		}
		return out, nil
	case string:
		out := []any{}
		for i := 0; i < len(x); {
			_, size := utf8.DecodeRuneInString(x[i:])
			if size == 1 && x[i] >= utf8.RuneSelf && i+3 <= len(x) { // a WTF-8 surrogate
				size = 3
			}
			out = append(out, x[i:i+size])
			i += size
		}
		return out, nil
	}
	return nil, &PyError{"TypeError"}
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
