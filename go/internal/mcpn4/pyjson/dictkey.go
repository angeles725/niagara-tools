package pyjson

import (
	"math"
	"math/big"
)

// DictKey is the identity of v as a key of a Python dict: equal keys collide as hash and ==
// make them (True is 1 is 1.0, False is 0 is -0.0), every NaN json.loads returns is the same
// object, and a list or dict cannot be a key (TypeError).
func DictKey(v any) (string, error) {
	switch x := v.(type) {
	case nil:
		return "None", nil
	case bool:
		if x {
			return "i1", nil
		}
		return "i0", nil
	case string:
		return "s" + x, nil
	case Number:
		if !x.IsFloat() {
			return "i" + x.String(), nil
		}
		f, _ := x.Float64()
		switch {
		case math.IsNaN(f):
			return "nan", nil
		case math.IsInf(f, 0) || f != math.Trunc(f):
			return "f" + FloatRepr(f), nil
		}
		n, _ := big.NewFloat(f).Int(nil)
		return "i" + n.String(), nil
	}
	return "", &Exception{Type: "TypeError"}
}
