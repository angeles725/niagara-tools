package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

// listing is one n4_navigate tree: the nodes the load answered, the same nodes with the declared
// defaults of the listed components (declared) and the types filter (nil: none).
type listing struct {
	loaded   *pyjson.Object
	declared *declared
	types    *typeFilter
}

// listedPaths are the paths whose children a listing depth levels deep shows: the loaded
// component and every node down to depth-1 levels below it.
func listedPaths(nodes *pyjson.Object, depth int) []string {
	paths := []string{""}
	for _, p := range nodes.Keys() {
		if p != "" && strings.Count(p, "/") <= depth-2 {
			paths = append(paths, p)
		}
	}
	return paths
}

// entries is _tree_entries: the child entries of path, levels deep. With types an entry is
// kept when the filter matches its type (matched) or a nested entry is kept (the path to it);
// has_children is never filtered, so a caller can still descend below the depth (#179 R3-003),
// and counts the components the station sent only. A declared property at its default is an
// entry marked default with its value; one whose contract types no default comes last.
func (l *listing) entries(path string, levels int) ([]any, error) {
	entries := []any{}
	nodes := l.declared.nodes
	kids := box.Children(nodes, path)
	for _, name := range kids.Keys() {
		node, childPath := box.Node(kids, name), childPath(path, name)
		kind, _ := node.Get("t")
		handle, _ := node.Get("h")
		has, err := hasComponentChildren(l.loaded, childPath)
		if err != nil {
			return nil, err
		}
		entry := obj("name", name, "type", kind, "handle", handle, "has_children", has)
		if l.declared.paths[childPath] {
			if err := l.declared.navDefault(entry, childPath); err != nil {
				return nil, err
			}
		}
		var nested []any
		if levels > 1 {
			if nested, err = l.entries(childPath, levels-1); err != nil {
				return nil, err
			}
			entry.Set("children", nested)
		}
		if l.types != nil {
			matched, err := l.types.match(kind)
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
	if l.types == nil {
		for _, name := range l.declared.untyped(path) {
			entries = append(entries, obj("name", name, "type", nil, "handle", nil, "has_children", false,
				"default", true, "value", nil, "note", noDefaultNote))
		}
	}
	return entries, nil
}
