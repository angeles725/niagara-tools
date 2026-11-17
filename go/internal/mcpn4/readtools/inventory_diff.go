package readtools

import (
	"slices"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The comparison of two inventory snapshots (station-audits S8), by slot path: components added,
// removed or retyped, slot values changed, and handles changed (informational: a handle moves when
// a component is re-created, not when it is edited). It reads only the two files.

// invDiff is what changed from snapshot a to snapshot b, each list in b's (or a's) path order.
type invDiff struct {
	added, removed, changed, typeChanged, handleChanged []*pyjson.Object
	unchanged                                           int
}

// diffInventory compares a with b.
func diffInventory(a, b *invSnapshot) *invDiff {
	d := &invDiff{}
	before := map[string]*invComponent{}
	for i := range a.components {
		before[a.components[i].path] = &a.components[i]
	}
	seen := map[string]bool{}
	for i := range b.components {
		after := &b.components[i]
		seen[after.path] = true
		old := before[after.path]
		switch {
		case old == nil:
			d.added = append(d.added, obj("path", after.path, "type", after.typ))
			continue
		case old.typ != after.typ:
			d.typeChanged = append(d.typeChanged, obj("path", after.path, "before", old.typ, "after", after.typ))
			continue
		}
		if old.handle != after.handle {
			d.handleChanged = append(d.handleChanged, obj("path", after.path, "before", old.handle, "after", after.handle))
		}
		changes := slotChanges(after.path, old.slots, after.slots)
		if len(changes) == 0 {
			d.unchanged++
		}
		d.changed = append(d.changed, changes...)
	}
	for _, c := range a.components {
		if !seen[c.path] {
			d.removed = append(d.removed, obj("path", c.path, "type", c.typ))
		}
	}
	return d
}

// slotChanges are the slots of one component whose value differs; a slot on one side only has
// null on the other (not sent: at its default, or no such slot).
func slotChanges(path string, before, after *pyjson.Object) []*pyjson.Object {
	names := slices.Clone(before.Keys())
	for _, n := range after.Keys() {
		if _, ok := before.Get(n); !ok {
			names = append(names, n)
		}
	}
	out := []*pyjson.Object{}
	for _, n := range names {
		b, _ := before.Get(n)
		a, _ := after.Get(n)
		if sameInvValue(b, a) {
			continue
		}
		out = append(out, obj("path", path, "slot", n, "before", b, "after", a))
	}
	return out
}

// sameInvValue says whether two recorded values are equal: both the same text, or both null.
func sameInvValue(x, y any) bool {
	xs, xok := x.(string)
	ys, yok := y.(string)
	if xok || yok {
		return xok && yok && xs == ys
	}
	return x == nil && y == nil
}

// counts are the sizes of every list of d, before any cap.
func (d *invDiff) counts() *pyjson.Object {
	return obj("added", len(d.added), "removed", len(d.removed), "changed", len(d.changed),
		"type_changed", len(d.typeChanged), "handle_changed", len(d.handleChanged), "unchanged", d.unchanged)
}

// reply is the n4_inventory_diff answer: each list cut to limit items (truncated says one was).
func (d *invDiff) reply(a, b *invSnapshot, limit int) *pyjson.Object {
	truncated := false
	capped := func(items []*pyjson.Object) []any {
		if len(items) > limit {
			truncated, items = true, items[:limit]
		}
		return list(items)
	}
	out := obj("a", a.name, "b", b.name, "counts", d.counts(), "added", capped(d.added),
		"removed", capped(d.removed), "changed", capped(d.changed), "type_changed", capped(d.typeChanged),
		"handle_changed", capped(d.handleChanged))
	out.Set("truncated", truncated)
	out.Set("notes", strs(inventoryNotes(a, b)))
	return out
}

// inventoryNotes say what makes the two sides not alike: another station, base, depth or slot
// choice, a snapshot cut at max_components, or one taken without the type contracts.
func inventoryNotes(a, b *invSnapshot) []string {
	notes := []string{}
	if a.station != b.station {
		notes = append(notes, "the snapshots come from different stations ("+a.station+", "+b.station+")")
	}
	if a.base != b.base || a.depth != b.depth {
		notes = append(notes, "the snapshots cover different subtrees (base or depth differ): added and removed include that difference")
	}
	if !slices.Equal(a.chosen, b.chosen) || (a.chosen == nil) != (b.chosen == nil) {
		notes = append(notes, "the snapshots record different slots (the slots argument differs): changed includes that difference")
	}
	if a.truncated || b.truncated {
		notes = append(notes, "a snapshot stopped at max_components: added and removed are not complete")
	}
	if a.contracts != b.contracts {
		notes = append(notes, "one snapshot was taken without the type contracts: a slot its type declares transient can show as changed")
	}
	return notes
}
