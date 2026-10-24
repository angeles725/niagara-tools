package readtools

import (
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// linkTypes is LINK_TYPES.
var linkTypes = map[string]bool{"baja:Link": true, "baja:ConversionLink": true}

// isLinkType is `kind in LINK_TYPES` for any decoded value.
func isLinkType(kind any) bool {
	s, ok := kind.(string)
	return ok && linkTypes[s]
}

// The Python exceptions a node of the wrong shape raises: only their type is shown.
var (
	attributeError = &pyjson.Exception{Type: "AttributeError"} // a str method on a non-str
	typeError      = &pyjson.Exception{Type: "TypeError"}      // an unhashable set member
)

// ordArg is _ord_arg: the "ord" argument (default RootOrd), a station ORD.
func ordArg(args *pyjson.Object) (string, error) {
	ord := RootOrd
	if v, ok := args.Get("ord"); ok {
		ord = v.(string) // validated against the schema
	}
	if !strings.HasPrefix(ord, "station:") {
		return "", toolError("ord must be a station ORD such as %s, got %s", pyjson.StrRepr(RootOrd), pyjson.StrRepr(ord))
	}
	return ord, nil
}

// intArg is args.get(key, def) for an integer the schema bounds.
func intArg(args *pyjson.Object, key string, def int) int {
	if v, ok := args.Get(key); ok {
		n, _ := v.(pyjson.Number).BigInt()
		return int(n.Int64())
	}
	return def
}

// isComponent is _is_component: a child with no simple value that is not a link or a Status
// complex. A type that is not a str, on a node without "v", is Python's AttributeError.
func isComponent(node *pyjson.Object) (bool, error) {
	if _, ok := node.Get("v"); ok {
		return false, nil
	}
	kind, ok := node.Get("t")
	if !ok {
		kind = ""
	}
	if isLinkType(kind) {
		return false, nil
	}
	s, ok := kind.(string)
	if !ok {
		return false, attributeError
	}
	return !strings.HasPrefix(s, "baja:Status"), nil
}

// hasComponentChildren is _has_component_children: any() stops at the first component.
func hasComponentChildren(nodes *pyjson.Object, path string) (bool, error) {
	kids := box.Children(nodes, path)
	for _, name := range kids.Keys() {
		if yes, err := isComponent(box.Node(kids, name)); yes || err != nil {
			return yes, err
		}
	}
	return false, nil
}

// typeIn is `kind in types` for the set of type specs: a list or dict kind is unhashable.
func typeIn(kind any, types map[string]bool) (bool, error) {
	switch k := kind.(type) {
	case string:
		return types[k], nil
	case []any, *pyjson.Object:
		return false, typeError
	}
	return false, nil
}

// treeEntries is _tree_entries: the child entries of path, levels deep. With types an entry is
// kept when its type is in the set (matched) or a nested entry is kept (the path to it);
// has_children is never filtered, so a caller can still descend below the depth (#179 R3-003).
func treeEntries(nodes *pyjson.Object, path string, levels int, types map[string]bool) ([]any, error) {
	entries := []any{}
	kids := box.Children(nodes, path)
	for _, name := range kids.Keys() {
		node, childPath := box.Node(kids, name), name
		if path != "" {
			childPath = path + "/" + name
		}
		kind, _ := node.Get("t")
		handle, _ := node.Get("h")
		has, err := hasComponentChildren(nodes, childPath)
		if err != nil {
			return nil, err
		}
		entry := obj("name", name, "type", kind, "handle", handle, "has_children", has)
		var nested []any
		if levels > 1 {
			if nested, err = treeEntries(nodes, childPath, levels-1, types); err != nil {
				return nil, err
			}
			entry.Set("children", nested)
		}
		if types != nil {
			matched, err := typeIn(kind, types)
			if err != nil {
				return nil, err
			}
			entry.Set("matched", matched)
			if !matched && len(nested) == 0 {
				continue
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

// typesArg is _types_arg: the "types" filter as given and as a set, or nil when absent or
// empty (no filter).
func typesArg(args *pyjson.Object) ([]any, map[string]bool, error) {
	v, _ := args.Get("types")
	if !pyjson.Truthy(v) {
		return nil, nil, nil
	}
	list := v.([]any) // validated against the schema
	set := map[string]bool{}
	for _, item := range list {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, nil, toolError("types must be a list of non-empty type specs such as "+
				"'bacnet:BacnetDevice', got %s", pyjson.Repr(list))
		}
		set[s] = true
	}
	return list, set, nil
}

// timedLoad is _timed_load: one LoadTree and its latency in whole milliseconds (retro
// 2026-10-02 D2), rounded half to even as Python's round().
func (t *Tools) timedLoad(ord string, depth int) (*pyjson.Object, int, error) {
	start := t.Ctx.Clock()
	nodes, err := t.Ctx.Session.Client.(Client).LoadTree(ord, depth, box.LoadOptions{})
	return nodes, int(math.RoundToEven((t.Ctx.Clock() - start) * 1000)), err
}

// Navigate is n4_navigate: the children of ord, depth levels deep, optionally kept by type.
func (t *Tools) Navigate(args *pyjson.Object) (*pyjson.Object, error) {
	ord, err := ordArg(args)
	if err != nil {
		return nil, err
	}
	depth := intArg(args, "depth", 1)
	list, types, err := typesArg(args)
	if err != nil {
		return nil, err
	}
	// one extra level so has_children is known for the deepest children returned
	nodes, elapsed, err := t.timedLoad(ord, depth+1)
	if err != nil {
		return nil, err
	}
	children, err := treeEntries(nodes, "", depth, types)
	if err != nil {
		return nil, err
	}
	out := obj("ord", ord, "children", children, "elapsed_ms", elapsed)
	if list != nil {
		out.Set("types", list)
	}
	return out, nil
}
