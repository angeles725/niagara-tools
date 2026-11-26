package boxstation

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// genericRefusal is what a live N4.14 station answers for an add it cannot process.
const genericRefusal = "ValueError: Unable to process request. Please contact your system administrator."

// unencodable is live finding 3 (2026-10-01): a "t": null node, or a Status carrying runtime
// facets (";" in its value), anywhere in an add body.
func unencodable(b map[string]any) bool {
	if t, ok := b["t"]; ok && t == nil {
		return true
	}
	if (b["n"] == "status" || b["t"] == "baja:Status") && strings.Contains(fmt.Sprint(valueOr(b["v"])), ";") {
		return true
	}
	subs, _ := b["s"].([]any)
	for _, s := range subs {
		if m, ok := s.(map[string]any); ok && unencodable(m) {
			return true
		}
	}
	return false
}

func valueOr(v any) any {
	if v == nil {
		return ""
	}
	return v
}

// sync applies one syncTo op: add (a), set (s), rename (r), reorder (o), set flags (f) or remove (v).
func (st *Station) sync(op map[string]any) (any, error) {
	name, _ := op["n"].(string)
	if strings.Contains(name, "//") {
		return nil, fmt.Errorf("ValueError: Illegal double slashes")
	}
	h, _ := op["h"].(string)
	parent := st.byHandle[h]
	if parent == nil {
		return nil, fmt.Errorf("KeyError: '%s'", h)
	}
	body, _ := op["b"].(map[string]any)
	switch op["nm"] {
	case "a":
		if !st.isComponent(body["t"]) {
			flags, _ := op["f"].(string)
			return st.addPlain(parent, name, body, flags)
		}
		subs, _ := body["s"].([]any)
		for _, s := range subs {
			if m, ok := s.(map[string]any); ok && st.isComponent(m["t"]) {
				return nil, errors.New(genericRefusal)
			}
		}
		if unencodable(body) {
			return nil, errors.New(genericRefusal)
		}
		if writables[parent.Type] && (name == "proxyExt" || strings.Contains(fmt.Sprint(body["t"]), "ProxyExt")) {
			return nil, fmt.Errorf("ValueError: Illegal child \"%v\" for parent \"%s\".", body["t"], parent.Type)
		}
		typ, _ := body["t"].(string)
		node := st.Component(parent, st.unique(parent, name), typ, "")
		st.fill(node, body)
		if writables[node.Type] { // the frozen slot, created with its parent
			st.Component(node, "proxyExt", "control:NullProxyExt", "")
		}
		if st.AfterAdd != nil {
			st.AfterAdd(st, node)
		}
		return []any{map[string]any{"id": "a", "nn": node.Name}}, nil
	case "s":
		parts := strings.Split(name, "/")
		node := parent
		for _, p := range parts[:len(parts)-1] {
			if node = node.Child(p); node == nil {
				return nil, fmt.Errorf("AttributeError: 'NoneType' object has no attribute 'child'")
			}
		}
		leaf := node.Child(parts[len(parts)-1])
		if leaf == nil {
			typ, _ := body["t"].(string)
			leaf = &Node{Name: parts[len(parts)-1], Type: typ}
			node.Children = append(node.Children, leaf)
		}
		st.fill(leaf, body)
		if st.AfterSet != nil {
			st.AfterSet(st, leaf)
		}
		return []any{}, nil
	case "r": // RenameOp: parent handle, old name "o", new name "n"
		old, _ := op["o"].(string)
		child := parent.Child(old)
		if child == nil {
			return nil, fmt.Errorf("KeyError: '%s'", old)
		}
		child.Name = name
		return []any{}, nil
	case "o": // ReorderOp: parent handle, the slot names joined with ";" in "o"
		return nil, reorder(parent, op["o"])
	case "f":
		return setFlags(parent, name, op)
	case "v":
		for i, c := range parent.Children {
			if c.Name == name {
				parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
				st.dropLinksFrom(c)
				return []any{}, nil
			}
		}
		return nil, fmt.Errorf("KeyError: '%s'", name)
	}
	return nil, fmt.Errorf("ValueError: unknown sync op %v", op["nm"])
}

// reorder applies a ReorderOp as the station does (baja ComponentSlotMap.reorder): the names must
// be every child once; the children then hold that order.
func reorder(parent *Node, list any) error {
	names := strings.Split(fmt.Sprint(valueOr(list)), ";")
	if len(names) != len(parent.Children) {
		return fmt.Errorf("ArrayIndexOutOfBoundsException: actual count %d != param count %d",
			len(parent.Children), len(names))
	}
	kids := make([]*Node, 0, len(names))
	for _, n := range names {
		c := parent.Child(n)
		if c == nil || slices.Contains(kids, c) {
			return fmt.Errorf("KeyError: '%s'", n)
		}
		kids = append(kids, c)
	}
	parent.Children = kids
	return nil
}
