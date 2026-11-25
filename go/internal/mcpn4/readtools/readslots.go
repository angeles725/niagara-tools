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
// and flags, an omitted child taking the type default. A baja:Status slot is not a complex:
// see statusEntry.
func slotEntry(nodes *pyjson.Object, name string) (*pyjson.Object, error) {
	node := box.Node(nodes, name)
	kind, _ := node.Get("t")
	display, _ := node.Get("d")
	value, _ := node.Get("v")
	entry := obj("name", name, "type", kind, "value", value, "status", nil)
	if display != nil {
		entry.Set("value_display", display)
	}
	if kind == "baja:Status" {
		return statusEntry(entry, value)
	}
	status, err := isStatusType(kind)
	if err != nil {
		return nil, err
	}
	if !status {
		if s, ok := value.(string); ok && (kind == "baja:Facets" || name == "facets") {
			entry.Set("facets_parsed", parseFacets(s))
		}
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

// parseFacets is a baja:Facets string ("units=u:fahrenheit;...|precision=i:1|maxOverrideDuration=r:3600000")
// as key to value, each value without its one-letter type prefix ("1", "3600000"). A part without
// "=" is left out.
func parseFacets(s string) *pyjson.Object {
	out := pyjson.NewObject()
	for _, part := range strings.Split(s, "|") {
		key, val, ok := strings.Cut(part, "=")
		if !ok || key == "" {
			continue
		}
		if len(val) >= 2 && val[1] == ':' {
			val = val[2:]
		}
		out.Set(key, val)
	}
	return out
}

// ReadSlots is n4_read_slots: every slot of ord, Status complexes decoded (depth 2: the slots
// and the value/status children of each Status slot), then the declared properties the load
// left out at their default (withDefaults), each marked default. An ord that names a property of
// a component (a struct such as alarm:AlarmTimestamps) lists that property's fields instead
// (property), or the property itself when it has none.
func (t *Tools) ReadSlots(args *pyjson.Object) (*pyjson.Object, error) {
	ord, err := ordArg(args)
	if err != nil {
		return nil, err
	}
	nodes, elapsed, err := t.timedLoad(ord, 2)
	if err != nil {
		return nil, err
	}
	d := t.withDefaults(nodes, []string{""})
	prop, err := t.property(ord, nodes, d)
	if err != nil {
		return nil, err
	}
	atDefault := false
	if prop != nil {
		nodes, d, atDefault = prop.nodes, prop.d, prop.atDefault
		elapsed += prop.elapsed
	}
	slots, err := listSlots(nodes, d, atDefault)
	if err != nil {
		return nil, err
	}
	if prop != nil && len(slots) == 0 {
		slots = append(slots, prop.self)
	}
	var notFound []any
	wanted, filtered := slotFilter(args)
	if filtered {
		slots, notFound = keepSlots(slots, wanted)
	}
	out := obj("ord", ord, "slots", slots, "elapsed_ms", elapsed)
	if filtered && len(notFound) > 0 {
		out.Set("slots_not_found", notFound)
	}
	if prop != nil {
		prop.describe(out, slots)
	}
	if note := d.noteText(); note != "" {
		out.Set("defaults_note", note)
	}
	return out, nil
}

// listSlots is the entry of every loaded slot under the root, then of each declared default
// the load left out; atDefault marks the loaded ones default too (the fields of a property that
// is itself at its declared default).
func listSlots(nodes *pyjson.Object, d *declared, atDefault bool) ([]any, error) {
	slots := []any{}
	for _, name := range box.Children(nodes, "").Keys() {
		if d.paths[name] {
			continue // listed by name only: its declared default is listed below
		}
		entry, err := slotEntry(nodes, name)
		if err != nil {
			return nil, err
		}
		if atDefault {
			entry.Set("default", true)
		}
		slots = append(slots, entry)
	}
	for _, name := range d.added[""] {
		if entry, err := d.defaultSlot(name, name); err != nil {
			d.skip(name, err) // one default the reader cannot decode does not fail the read
		} else {
			slots = append(slots, entry)
		}
	}
	return slots, nil
}

// statusEntry completes the entry of a slot whose type is baja:Status itself: a simple whose
// value is the encoded status (omitted when "0"), with no value/status children to read. The
// status is that value, status_flags names the bits set and status_ok is false when any is.
func statusEntry(entry *pyjson.Object, value any) (*pyjson.Object, error) {
	if value == nil {
		value = "0"
	}
	flags, err := box.StatusFlags(value)
	if err != nil {
		return nil, err
	}
	_, null, _ := box.ParseStatus(value)
	names := make([]any, len(flags))
	for i, f := range flags {
		names[i] = f
	}
	entry.Set("value", value)
	entry.Set("status", value)
	entry.Set("status_ok", len(flags) == 0)
	entry.Set("status_null", null)
	entry.Set("status_flags", names)
	return entry, nil
}
