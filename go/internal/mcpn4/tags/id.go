// Package tags is the codec of Niagara tags as a BOX session sees them: a direct tag is a
// dynamic property with the metadata flag named by its escaped id ("hs:point" is the slot
// "hs$3apoint"), a tag group or relation is a baja:Relation slot, and implied tags come from the
// session handler's impliedTags call keyed id + type letter. It holds data and pure functions
// only; the tools that read and write tags compose it.
package tags

import (
	"regexp"
	"strings"
)

// escapedColon is SlotPath.escape(":"): the separator of namespace and name in a tag slot name.
const escapedColon = "$3a"

// idPattern is a tag id the tools accept: namespace:name, each a NEQL word of letters, digits
// and underscore, so the slot name needs no other escape and a NEQL query can name it.
var idPattern = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*):([A-Za-z0-9_]+)$`)

// ID is one tag id split in its parts.
type ID struct{ Namespace, Name string }

// String is the qualified name, namespace:name.
func (id ID) String() string { return id.Namespace + ":" + id.Name }

// Slot is the slot name that holds the tag on a component.
func (id ID) Slot() string { return id.Namespace + escapedColon + id.Name }

// ParseID is the id of text, ok false unless it is namespace:name of NEQL words.
func ParseID(text string) (ID, bool) {
	m := idPattern.FindStringSubmatch(text)
	if m == nil {
		return ID{}, false
	}
	return ID{m[1], m[2]}, true
}

// FromSlot is the qualified tag id a slot name holds ("hs$3apoint" is "hs:point"); a name
// without an escaped colon is its own id (a tag without namespace).
func FromSlot(slot string) string {
	return strings.Replace(slot, escapedColon, ":", 1)
}

// IsTagSlot is BajaScript ComponentTags.isTaggableSlot over one loaded node: a property ("nm"
// "p") that is no component (no handle), not transient, with the metadata flag, whose name holds
// at most one escaped colon.
func IsTagSlot(nm, name, flags string, hasHandle bool) bool {
	if nm != "p" || hasHandle || !strings.Contains(flags, "m") || strings.Contains(flags, "t") {
		return false
	}
	return strings.Count(name, escapedColon) <= 1
}
