package plan

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Resolve returns nodes with slot present. A listed slot leaves nodes as they are (the listed
// value's type is authoritative). The station omits a frozen property at its type default, so an
// omitted slot is looked up in the type definition of its owner (the root, or the owner of a
// nested path, typed the same way when it is omitted too): when declared, the result is a copy
// of nodes holding the declared default as slot's node (and its children), so the type check,
// the previous value and the read-back see what the station holds. A client without TypeLoader
// or a slot no definition declares leaves nodes unchanged, and the caller refuses as before.
func Resolve(c Client, nodes *pyjson.Object, slot string) (*pyjson.Object, error) {
	if box.Node(nodes, slot) != nil {
		return nodes, nil
	}
	tl, ok := c.(TypeLoader)
	if !ok {
		return nodes, nil
	}
	def, err := declaredNode(tl, nodes, slot)
	if err != nil || def == nil {
		return nodes, err
	}
	out := pyjson.NewObject()
	for _, k := range nodes.Keys() {
		v, _ := nodes.Get(k)
		out.Set(k, v)
	}
	out.Set(slot, def)
	kids, _ := field(def, "s").([]any)
	for _, k := range kids {
		if kid, ok := k.(*pyjson.Object); ok {
			if n, ok := field(kid, "n").(string); ok {
				out.Set(slot+"/"+n, kid)
			}
		}
	}
	return out, nil
}

// declaredNode is the declared default of path: its owner's type from the listed owner node, or
// from the owner's own declared default when the owner is omitted as well.
func declaredNode(tl TypeLoader, nodes *pyjson.Object, path string) (*pyjson.Object, error) {
	owner, name := "", path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		owner, name = path[:i], path[i+1:]
	}
	ownerNode := box.Node(nodes, owner)
	if ownerNode == nil && owner != "" {
		var err error
		if ownerNode, err = declaredNode(tl, nodes, owner); err != nil || ownerNode == nil {
			return nil, err
		}
	}
	ownerType, _ := field(ownerNode, "t").(string)
	if ownerType == "" {
		return nil, nil
	}
	return DeclaredDefault(tl, ownerType, name)
}
