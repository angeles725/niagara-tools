package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// items is iter(v) of a decoded value: a list's items, a string's code points, a dict's keys;
// anything else is Python's TypeError.
func items(v any) ([]any, error) {
	switch x := v.(type) {
	case []any:
		return x, nil
	case string:
		var out []any
		for i := 0; i < len(x); {
			_, n := pyjson.DecodeRune(x[i:])
			out = append(out, x[i:i+n])
			i += n
		}
		return out, nil
	case *pyjson.Object:
		out := make([]any, 0, x.Len())
		for _, k := range x.Keys() {
			out = append(out, k)
		}
		return out, nil
	}
	return nil, &plan.PyError{Type: "TypeError", Msg: "'" + plan.TypeName(v) + "' object is not iterable"}
}

// orEmpty is `v or []`.
func orEmpty(v any) any {
	if pyjson.Truthy(v) {
		return v
	}
	return []any{}
}

// dict is v as a dict whose attribute attr is called: anything else is Python's AttributeError.
func dict(v any, attr string) (*pyjson.Object, error) {
	if o, ok := v.(*pyjson.Object); ok {
		return o, nil
	}
	return nil, &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(v) + "' object has no attribute '" + attr + "'"}
}

// need is o[k]: a missing key is Python's KeyError.
func need(o *pyjson.Object, k string) (any, error) {
	if v, ok := o.Get(k); ok {
		return v, nil
	}
	return nil, &plan.PyError{Type: "KeyError", Msg: pyjson.StrRepr(k)}
}

// pair is the identity of the tuple (a, b) as a dict key; an unhashable item is TypeError.
func pair(a, b any) (string, error) {
	ka, err := pyjson.DictKey(a)
	kb, err2 := pyjson.DictKey(b)
	if err != nil || err2 != nil {
		bad := a
		if err == nil {
			bad = b
		}
		t := plan.TypeName(bad)
		return "", &plan.PyError{Type: "TypeError", Msg: "cannot use 'tuple' as a dict key (unhashable type: '" + t + "')"}
	}
	return strconv.Itoa(len(ka)) + ":" + ka + kb, nil
}
