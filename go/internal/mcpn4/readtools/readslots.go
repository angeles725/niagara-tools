package readtools

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// isStatusType is (kind or "").startswith("baja:Status"): a truthy non-str is AttributeError.
func isStatusType(kind any) (bool, error) {
	if !pyjson.Truthy(kind) {
		return false, nil
	}
	s, ok := kind.(string)
	if !ok {
		return false, attributeError
	}
	return strings.HasPrefix(s, "baja:Status"), nil
}

// slotEntry is _slot_entry: one slot's type, value and display; a Status slot adds its status
// and flags, an omitted child taking the type default.
func slotEntry(nodes *pyjson.Object, name string) (*pyjson.Object, error) {
	node := box.Node(nodes, name)
	kind, _ := node.Get("t")
	display, _ := node.Get("d")
	value, _ := node.Get("v")
	entry := obj("name", name, "type", kind, "value", value, "status", nil)
	if display != nil {
		entry.Set("value_display", display)
	}
	status, err := isStatusType(kind)
	if err != nil {
		return nil, err
	}
	if !status {
		if value == nil && display != nil {
			// a complex the reader does not decode (FlexAddress, BacnetAddress, ...): its
			// display string beats a bare None (retro 2026-10-02 D4)
			entry.Set("value", display)
		}
		return entry, nil
	}
	value, st, err := box.StatusValue(nodes, name)
	if ve := (*pyjson.ValueError)(nil); errors.As(err, &ve) {
		// a Status type without a known default (e.g. StatusString)
		kids := box.Children(nodes, name)
		value, st = box.ChildValue(kids, "value", nil), box.ChildValue(kids, "status", "0")
	}
	entry.Set("value", value)
	entry.Set("status", st)
	ok, null, err := box.ParseStatus(st)
	if err != nil {
		return nil, err
	}
	entry.Set("status_ok", ok)
	entry.Set("status_null", null)
	return entry, nil
}

// ReadSlots is n4_read_slots: every slot of ord, Status complexes decoded (depth 2: the slots
// and the value/status children of each Status slot).
func (t *Tools) ReadSlots(args *pyjson.Object) (*pyjson.Object, error) {
	ord, err := ordArg(args)
	if err != nil {
		return nil, err
	}
	nodes, elapsed, err := t.timedLoad(ord, 2)
	if err != nil {
		return nil, err
	}
	slots := []any{}
	for _, name := range box.Children(nodes, "").Keys() {
		entry, err := slotEntry(nodes, name)
		if err != nil {
			return nil, err
		}
		slots = append(slots, entry)
	}
	return obj("ord", ord, "slots", slots, "elapsed_ms", elapsed), nil
}
