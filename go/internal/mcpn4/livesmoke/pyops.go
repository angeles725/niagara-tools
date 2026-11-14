package livesmoke

import (
	"fmt"
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// SmokeError means the run cannot continue (server gone, protocol error, tool error). A step
// that fails with one records its message as the detail.
type SmokeError struct{ Msg string }

func (e *SmokeError) Error() string { return e.Msg }

func smokef(format string, a ...any) error { return &SmokeError{fmt.Sprintf(format, a...)} }

// pyError is the Python exception a malformed server reply raised in live_smoke.py (KeyError,
// TypeError, AttributeError, IndexError): a step records it as "<Type>: <message>".
type pyError struct{ Type, Msg string }

func (e *pyError) Error() string { return e.Type + ": " + e.Msg }

// typeName is type(v).__name__ for a decoded JSON value.
func typeName(v any) string {
	switch x := v.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case pyjson.Number:
		if x.IsFloat() {
			return "float"
		}
		return "int"
	case []any:
		return "list"
	case *pyjson.Object:
		return "dict"
	}
	return fmt.Sprintf("%T", v)
}

// item is v[key] for a str key.
func item(v any, key string) (any, error) {
	switch x := v.(type) {
	case *pyjson.Object:
		if got, ok := x.Get(key); ok {
			return got, nil
		}
		return nil, &pyError{"KeyError", pyjson.StrRepr(key)}
	case []any:
		return nil, &pyError{"TypeError", "list indices must be integers or slices, not str"}
	case string:
		return nil, &pyError{"TypeError", "string indices must be integers, not 'str'"}
	}
	return nil, &pyError{"TypeError", fmt.Sprintf("'%s' object is not subscriptable", typeName(v))}
}

// index is v[i] for an int index.
func index(v any, i int) (any, error) {
	switch x := v.(type) {
	case []any:
		if i < len(x) {
			return x[i], nil
		}
		return nil, &pyError{"IndexError", "list index out of range"}
	case string:
		if i < len(x) {
			return x[i : i+1], nil
		}
		return nil, &pyError{"IndexError", "string index out of range"}
	case *pyjson.Object:
		return nil, &pyError{"KeyError", fmt.Sprint(i)}
	}
	return nil, &pyError{"TypeError", fmt.Sprintf("'%s' object is not subscriptable", typeName(v))}
}

// get is v.get(key) (nil when absent); only a dict has .get.
func get(v any, key string) (any, error) {
	o, ok := v.(*pyjson.Object)
	if !ok {
		return nil, &pyError{"AttributeError", fmt.Sprintf("'%s' object has no attribute 'get'", typeName(v))}
	}
	got, _ := o.Get(key)
	return got, nil
}

// getOr is v.get(key, def).
func getOr(v any, key string, def any) (any, error) {
	if o, ok := v.(*pyjson.Object); ok {
		if got, found := o.Get(key); found {
			return got, nil
		}
		return def, nil
	}
	return get(v, key)
}

// iterate is `for x in v`: a list's items, a dict's keys, a str's characters.
func iterate(v any) ([]any, error) {
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
		for _, r := range x {
			out = append(out, string(r))
		}
		return out, nil
	}
	return nil, &pyError{"TypeError", fmt.Sprintf("'%s' object is not iterable", typeName(v))}
}

// equalsTrue is `v == True`: True, the int 1 and the float 1.0.
func equalsTrue(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case pyjson.Number:
		if x.IsFloat() {
			f, _ := x.Float64()
			return f == 1
		}
		i, _ := x.BigInt()
		return i.Cmp(big.NewInt(1)) == 0
	}
	return false
}

// numberEquals is `v == n` for an int n (a reply id).
func numberEquals(v any, n int) bool {
	x, ok := v.(pyjson.Number)
	if !ok {
		return false
	}
	if x.IsFloat() {
		f, _ := x.Float64()
		return f == float64(n)
	}
	i, _ := x.BigInt()
	return i.Cmp(big.NewInt(int64(n))) == 0
}
