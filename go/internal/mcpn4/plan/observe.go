package plan

import (
	"errors"
	"fmt"
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

// NewName is Name for a component about to be created, plus the display names a station can only
// store escaped: printable ASCII other than '/', '|', '$' and "..", escaped the way SlotPath.escape
// does (a leading digit and every character outside letters, digits and underscore become $ and
// two hex digits: "2012" is stored as "$32012", "Room Temp" as "Room$20Temp"). It returns the
// stored name.
func NewName(label string, value any) (string, error) {
	if s, err := Name(label, value); err == nil {
		return s, nil
	}
	s, ok := value.(string)
	if !ok || s == "" || s == ".." || strings.ContainsAny(s, "/|$") {
		return Name(label, value)
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch < 0x20 || ch > 0x7e {
			return Name(label, value)
		}
		letter := ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
		digit := ch >= '0' && ch <= '9'
		if letter || (digit && i > 0) {
			b.WriteByte(ch)
		} else {
			fmt.Fprintf(&b, "$%02x", ch)
		}
	}
	return b.String(), nil
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
		return nil, nil, nameUnloadedProgram(ord, err)
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

// msgProgramUnloaded is the cause a station's refusal of a path below a Spyder's ControlProgram
// hides: the program unloads on its own and the station then answers with the bare path as the
// error (retro mcpn4-spyder-commissioning-over-box Δ4, live 2026-11-18).
const msgProgramUnloaded = "; this path is below a Spyder ControlProgram, which unloads on its own and then answers a path " +
	"below it with only the path: load it again (run n4_invoke_action loadControlProgram on the Spyder) and repeat the call"

// nameUnloadedProgram is err with the unloaded-program cause added when it is the station's own
// refusal of a path below a ControlProgram slot; any other error is returned as it was.
func nameUnloadedProgram(ord string, err error) error {
	var be *box.Error
	if !errors.As(err, &be) || !be.Station || !strings.Contains(ord, "/ControlProgram/") {
		return err
	}
	named := *be
	named.Msg += msgProgramUnloaded
	return &named
}
