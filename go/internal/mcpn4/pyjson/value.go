// Package pyjson reproduces the parts of CPython's json module that the mcp-n4 server relies on:
// json.dumps with the default and the compact separators, sort_keys, ensure_ascii and no HTML
// escaping; json.loads with the int/float distinction kept; and repr(float) / float(int).
//
// Values are the Go types nil, bool, string, Number, []any and *Object (an insertion-ordered
// dict). A string holds UTF-8, or WTF-8 when json.loads decoded a lone surrogate escape such as
// "\ud800": the encoder writes those three bytes back as the same \udxxx escape.
package pyjson

import (
	"math/big"
	"strconv"
)

// Object is a Python dict with string keys: iteration follows first insertion, and setting an
// existing key replaces its value in place. The zero value is not usable; call NewObject.
type Object struct {
	keys []string
	vals map[string]any
}

// NewObject returns an empty Object.
func NewObject() *Object { return &Object{vals: map[string]any{}} }

// Set stores v under k, keeping k's original position when it already exists.
func (o *Object) Set(k string, v any) {
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
}

// Get returns the value under k and whether it exists.
func (o *Object) Get(k string) (any, bool) {
	v, ok := o.vals[k]
	return v, ok
}

// Keys returns the keys in insertion order (a copy).
func (o *Object) Keys() []string { return append([]string(nil), o.keys...) }

// Len returns the number of keys.
func (o *Object) Len() int { return len(o.keys) }

// Number is a decoded JSON number with Python's kind: an int (arbitrary precision, kept as its
// literal) or a float. A JSON bool is never a Number.
type Number struct {
	lit     string // int only: canonical decimal literal
	isFloat bool
	f       float64
}

// IntLiteral returns the int whose decimal literal is lit (an optional '-' then digits without
// leading zeros, as JSON writes it). "-0" becomes 0, as int("-0") does.
func IntLiteral(lit string) Number {
	if lit == "-0" {
		lit = "0"
	}
	return Number{lit: lit}
}

// FloatNumber returns the float f.
func FloatNumber(f float64) Number { return Number{isFloat: true, f: f} }

// IsFloat reports whether n is a Python float (else it is an int).
func (n Number) IsFloat() bool { return n.isFloat }

// BigInt returns the value of an int, or false for a float.
func (n Number) BigInt() (*big.Int, bool) {
	if n.isFloat {
		return nil, false
	}
	return new(big.Int).SetString(n.lit, 10)
}

// String returns the JSON text json.dumps writes for n.
func (n Number) String() string {
	if n.isFloat {
		return floatJSON(n.f)
	}
	return n.lit
}

// Exception is a Python exception the server reports by its type only ("internal error
// (<Type>)"), such as the AttributeError a str method raises on a value that is not a str.
type Exception struct{ Type string }

func (e *Exception) Error() string { return e.Type }

// PyType is the Python class name.
func (e *Exception) PyType() string { return e.Type }

// OverflowError is Python's OverflowError from float(int) for an int beyond the float range.
type OverflowError struct{}

func (*OverflowError) Error() string { return "int too large to convert to float" }

// Float64 returns float(n). An int is rounded half-to-even; one that rounds past the largest
// float fails with *OverflowError, as float(10**400) does.
func (n Number) Float64() (float64, error) {
	if n.isFloat {
		return n.f, nil
	}
	f, err := strconv.ParseFloat(n.lit, 64)
	if err != nil {
		return 0, &OverflowError{}
	}
	return f, nil
}

// Truthy is bool(v) in Python for a decoded value: None, False, zero, "", [] and {} are false.
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case Number:
		if x.IsFloat() {
			f, _ := x.Float64()
			return f != 0
		}
		i, _ := x.BigInt()
		return i.Sign() != 0
	case int:
		return x != 0
	case []any:
		return len(x) > 0
	case *Object:
		return x.Len() > 0
	}
	return true
}
