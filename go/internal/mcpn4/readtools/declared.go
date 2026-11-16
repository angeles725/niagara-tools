package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Declared properties at their default (lab retro 2026-11-15 row 1): a BOX load omits a frozen
// property still at its type default, so a fresh component reads no slots at all. A listing adds
// each property its type contract declares (the registry's loadTypes, read once per session)
// that the load left out, as the contract's default value (plan.Resolve, the same default the
// writes use); a contract that states no default gives no value, never an invented one.

// The notes of a listing whose defaults are incomplete: the property's own, and the reply's.
const (
	noDefaultNote       = "at its declared default, but the type contract gives no default value (a component default is not sent): read it in Workbench"
	contractsUnreadNote = "the station's type registry could not be read (%s): declared properties at their default are not listed"
	notDescribedNote    = "the station did not describe type %s: its declared properties at their default are not listed"
	unknownTypeNote     = "the station does not know type %s: its declared properties at their default are not listed"
)

// declaredProps is the property names spec and its supertypes declare, inherited first (a name a
// subtype declares again keeps its first place), and a note when the walk met a type the station
// did not describe or does not know. An entry {} declares nothing and ends the walk.
func declaredProps(types *pyjson.Object, spec string) (names []string, note string) {
	var chain []*pyjson.Object
	for i := 0; i < maxSupertypes && spec != ""; i++ {
		e := box.Node(types, spec)
		if e == nil {
			note = fmt.Sprintf(notDescribedNote, spec)
			break
		}
		if v, _ := e.Get("isErr"); v == true {
			note = fmt.Sprintf(unknownTypeNote, spec)
			break
		}
		chain = append(chain, e)
		spec = box.Supertype(types, spec)
	}
	seen := map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		slots, _ := chain[i].Get("c")
		list, _ := slots.([]any)
		for _, s := range list {
			slot, _ := s.(*pyjson.Object)
			if slot == nil {
				continue
			}
			kind, _ := slot.Get("st")
			name, isStr := slotName(slot)
			if kind == "p" && isStr && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names, note
}

func slotName(slot *pyjson.Object) (string, bool) {
	v, _ := slot.Get("n")
	s, ok := v.(string)
	return s, ok && s != ""
}

// nodeType is the type spec of the node at path, "" when it has none.
func nodeType(nodes *pyjson.Object, path string) string {
	n := box.Node(nodes, path)
	if n == nil {
		return ""
	}
	v, _ := n.Get("t")
	s, _ := v.(string)
	return s
}

func childPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "/" + name
}
