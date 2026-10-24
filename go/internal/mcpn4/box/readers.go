package box

import (
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The readers of a LoadTree result (box.py "readers"): nodes maps a slot path ("" the loaded
// component) to its node, in load order.

// defaultValue is _DEFAULT_VALUE: the value a station omits from a Status* slot.
var defaultValue = map[string]string{"baja:StatusBoolean": "false", "baja:StatusNumeric": "0.0"}

// Node is nodes[path] (nil when absent): every node LoadTree returns is an object.
func Node(nodes *pyjson.Object, path string) *pyjson.Object {
	v, _ := nodes.Get(path)
	n, _ := v.(*pyjson.Object)
	return n
}

// Children is box.children: the direct children of path as {name: node}, in load order.
func Children(nodes *pyjson.Object, path string) *pyjson.Object {
	prefix := ""
	if path != "" {
		prefix = path + "/"
	}
	out := pyjson.NewObject()
	for _, key := range nodes.Keys() {
		if key != "" && key != path && strings.HasPrefix(key, prefix) && !strings.Contains(key[len(prefix):], "/") {
			v, _ := nodes.Get(key)
			out.Set(key[len(prefix):], v)
		}
	}
	return out
}

// ChildValue is kids.get(name, {}).get("v", def) on a Children result.
func ChildValue(kids *pyjson.Object, name string, def any) any {
	if n := Node(kids, name); n != nil {
		if v, ok := n.Get("v"); ok {
			return v
		}
	}
	return def
}

// StatusValue is box.status_value: the value and status of a StatusBoolean/StatusNumeric slot,
// an omitted child taking the type default; any other type is a ValueError.
func StatusValue(nodes *pyjson.Object, path string) (value, status any, err error) {
	kind, _ := Node(nodes, path).Get("t")
	name, _ := kind.(string)
	def, ok := defaultValue[name]
	if !ok {
		return nil, nil, &pyjson.ValueError{Msg: path + " is not a StatusBoolean/StatusNumeric"}
	}
	kids := Children(nodes, path)
	return ChildValue(kids, "value", def), ChildValue(kids, "status", "0"), nil
}

// ParseStatus is box.parse_status without the bits and facets no caller reads: ok is bits == 0
// and null is bit 0x40 of the hex bits before ";" (a falsy v is "0"). A truthy non-string is
// Python's AttributeError; bits int() cannot read are its ValueError.
func ParseStatus(v any) (ok, null bool, err error) {
	s, isStr := v.(string)
	switch {
	case !pyjson.Truthy(v):
		s = "0"
	case !isStr:
		return false, false, &pyjson.Exception{Type: "AttributeError"}
	}
	bitsStr, _, _ := strings.Cut(s, ";")
	if bitsStr == "" {
		bitsStr = "0"
	}
	bits, parsed := pyIntBase16(bitsStr)
	if !parsed {
		r := []rune(pyjson.StrRepr(bitsStr))
		return false, false, &pyjson.ValueError{Msg: "invalid literal for int() with base 16: " + string(r[:min(len(r), 200)])}
	}
	return bits.Sign() == 0, bits.Bit(6) == 1, nil
}

// pyIntBase16 is int(s, 16): outer whitespace, a sign, an optional 0x (then one optional "_"),
// hex digits with single "_" between them; a Unicode decimal digit reads as its ASCII digit.
func pyIntBase16(s string) (*big.Int, bool) {
	s = strings.TrimFunc(s, pyjson.IsSpace)
	neg := strings.HasPrefix(s, "-")
	if neg || strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = strings.TrimPrefix(s[2:], "_")
	}
	var digits strings.Builder
	under := true // no "_" first, none doubled
	for _, r := range s {
		switch {
		case r == '_' && !under:
			under = true
			continue
		case r < utf8.RuneSelf && strings.ContainsRune("0123456789abcdefABCDEF", r):
		case unicode.Is(unicode.Nd, r):
			r = '0' + pyjson.DecimalValue(r)
		default:
			return nil, false
		}
		under = false
		digits.WriteRune(r)
	}
	n, ok := new(big.Int).SetString(digits.String(), 16)
	if under || !ok { // trailing "_", no digits
		return nil, false
	}
	if neg {
		n.Neg(n)
	}
	return n, true
}
