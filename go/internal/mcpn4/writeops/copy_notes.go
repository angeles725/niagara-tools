package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const notCopiedNote = "Not copied (flags r readonly, t transient, d default-on-clone, R remove-on-clone): "

// strippedNode is a node the copy leaves out: its path below the source and the flags that
// decided it; readonly marks a frozen slot declared readonly-only holding a value off its default.
type strippedNode struct {
	path, flags string
	readonly    bool
}

// strippedNotes is one note listing every stripped node, then one per readonly-only slot whose
// value is lost; none when nothing was stripped.
func strippedNotes(list []strippedNode) []any {
	if len(list) == 0 {
		return nil
	}
	items, notes := []string{}, []any{}
	for _, s := range list {
		items = append(items, s.path+" ("+s.flags+")")
	}
	notes = append(notes, notCopiedNote+strings.Join(items, ", "))
	for _, s := range list {
		if s.readonly {
			notes = append(notes, "readonly slot holds a non-default value and is not copied: "+s.path)
		}
	}
	return notes
}

// sameValue is a node equal to the declared default: its value, and each child's by name.
func sameValue(node, def *pyjson.Object) bool {
	if def == nil || !box.PyEqual(get(node, "v"), get(def, "v")) {
		return false
	}
	names := map[string]bool{}
	for _, o := range []*pyjson.Object{node, def} {
		kids, _ := get(o, "s").([]any)
		for _, k := range kids {
			kid, ok := k.(*pyjson.Object)
			if !ok {
				return false
			}
			names[name(kid)] = true
		}
	}
	for n := range names {
		a, b := kidNamed(node, n), kidNamed(def, n)
		if a == nil || b == nil || !sameValue(a, b) {
			return false
		}
	}
	return true
}
