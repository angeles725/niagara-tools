package plan

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Observe is _observe: the current value of slot in the form Coerce returns.
func Observe(nodes *pyjson.Object, slot, valueType string) (any, error) {
	if IsStatusType(valueType) {
		if box.Node(nodes, slot) == nil {
			return nil, &PyError{Type: "KeyError", Msg: pyjson.StrRepr(slot)} // nodes[slot]
		}
		value, status, err := box.StatusValue(nodes, slot)
		if err != nil {
			return nil, err
		}
		var v any = box.PyEqual(value, "true")
		if strings.HasSuffix(valueType, "Numeric") {
			if v, err = Float(value); err != nil {
				return nil, err
			}
		}
		return obj("value", v, "status", status), nil
	}
	var raw any // a plain slot at its type default is omitted
	if n := box.Node(nodes, slot); n != nil {
		raw, _ = n.Get("v")
	}
	switch valueType {
	case "baja:Double":
		if !pyjson.Truthy(raw) {
			raw = "0.0"
		}
		return Float(raw)
	case "baja:Boolean":
		return box.PyEqual(raw, "true"), nil
	}
	if !pyjson.Truthy(raw) {
		return "", nil
	}
	return raw, nil
}

// Float is float(v) for a decoded value, with Python's errors.
func Float(v any) (float64, error) {
	switch x := v.(type) {
	case string:
		if f, ok := pyjson.ParseFloat(x); ok {
			return f, nil
		}
		return 0, &pyjson.ValueError{Msg: "could not convert string to float: " + pyjson.StrRepr(x)}
	case pyjson.Number:
		return x.Float64()
	case float64:
		return x, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	}
	return 0, &PyError{Type: "TypeError",
		Msg: "float() argument must be a string or a real number, not '" + TypeName(v) + "'"}
}

// Name is _name: value as one plain slot name, else the refusal naming label.
func Name(label string, value any) (string, error) {
	s, ok := value.(string)
	if !ok || !namePattern.MatchString(s) {
		return "", ToolError(label + " must be a plain slot name (letters, digits, underscore), got " +
			pyjson.Repr(value))
	}
	return s, nil
}

// escapedNamePattern is a slot name as a station stores it: letters, digits, underscore and
// SlotPath escapes ($ and two hex digits, e.g. $20 for a space).
var escapedNamePattern = regexp.MustCompile(`^(?:[A-Za-z0-9_]|\$[0-9A-Fa-f]{2})+$`)

// ExistingName is Name for a slot that already exists on the station: it also accepts the
// station's escapes, so a component named with a space ("Numeric$20Writable40001") can be read,
// copied or removed. New names stay plain (Name).
func ExistingName(label string, value any) (string, error) {
	s, ok := value.(string)
	if ok && (namePattern.MatchString(s) || escapedNamePattern.MatchString(s)) {
		return s, nil
	}
	// The refusal keeps Name's wording so the Python oracle comparison still matches it.
	return Name(label, value)
}

// Handle is _handle: load ord and return its handle and nodes; no handle is a refusal.
func Handle(c Client, ord string, depth int) (any, *pyjson.Object, error) {
	nodes, err := c.LoadTree(ord, depth, box.LoadOptions{})
	if err != nil {
		return nil, nil, err
	}
	var h any
	if root := box.Node(nodes, ""); root != nil {
		h, _ = root.Get("h")
	}
	if !pyjson.Truthy(h) {
		return nil, nil, ToolError("station returned no handle for " + ord)
	}
	return h, nodes, nil
}
