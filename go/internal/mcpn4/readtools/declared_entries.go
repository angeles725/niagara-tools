package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// defaultSlot is the n4_read_slots entry of a declared property at its default, full its path in
// d.nodes: the slot entry of the declared default, marked default; with no stated default value,
// value null and a note.
func (d *declared) defaultSlot(full, name string) (*pyjson.Object, error) {
	if box.Node(d.nodes, full) == nil { // the contract types no default
		return obj("name", name, "type", nil, "value", nil, "status", nil, "default", true, "note", noDefaultNote), nil
	}
	entry, err := slotEntry(d.nodes, full)
	if err != nil {
		return nil, err
	}
	entry.Set("name", name)
	entry.Set("default", true)
	if !d.states(full) {
		entry.Set("value", nil)
		entry.Set("note", noDefaultNote)
	}
	return entry, nil
}

// navDefault completes the n4_navigate entry of a declared property at its default: marked
// default, with the default's value, or value null and a note when the contract states none.
func (d *declared) navDefault(entry *pyjson.Object, full string) error {
	entry.Set("default", true)
	if !d.states(full) {
		entry.Set("value", nil)
		entry.Set("note", noDefaultNote)
		return nil
	}
	slot, err := slotEntry(d.nodes, full)
	if err != nil {
		return err
	}
	value, _ := slot.Get("value")
	entry.Set("value", value)
	return nil
}

// states reports whether the declared default at full carries a value: its own or its children's
// (a Status or struct default). A component default the station does not send ({t, d, nc}) has none.
func (d *declared) states(full string) bool {
	n := box.Node(d.nodes, full)
	if n == nil {
		return false
	}
	if _, ok := n.Get("v"); ok {
		return true
	}
	return box.Children(d.nodes, full).Len() > 0
}

// untyped is the names added under path whose contract types no default: no node holds them,
// so a listing that walks nodes adds them on its own.
func (d *declared) untyped(path string) []string {
	var out []string
	for _, name := range d.added[path] {
		if box.Node(d.nodes, childPath(path, name)) == nil {
			out = append(out, name)
		}
	}
	return out
}
