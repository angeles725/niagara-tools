package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A runtime slot is one the station never persists, so its value can change between a dry run
// and its execute by timing alone (retro 2026-11-16 row M15: an alarm extension's
// timeInCurrentState, BAlarmSourceExt flags 3 = readonly|transient, ticks every second). It is a
// frozen slot whose DECLARED flags hold t transient, or any slot whose instance flags hold t.
// Readonly alone is NOT runtime: Niagara marks a frozen struct readonly to mean "not replaceable"
// while its children are edited in place (BDateSchedule.yearSchedule, BDailySchedule.day), and
// persisted readonly state (alarmState, toOffnormalTimes) is a real change worth a new dry run.
// So this rule is its own, not computedFlags (keepsOutput's "rt" rule for an out).
//
// A runtime slot stays in the snapshot the journal keeps and a rollback re-adds, as before; only
// its value, and the values below it, leave the plan the token hashes, where it is marked
// "runtime": true. Names and types stay, so a slot or child added, removed or retyped still
// changes the hash, and so does any configuration value.
const runtimeFlags = "t"

// runtimeSlot says whether child, loaded under node, is a runtime slot. A contract that cannot be
// read leaves only the instance rule: a slot it cannot classify stays hashed (the safe side).
func (s *snapshot) runtimeSlot(node, child *pyjson.Object) bool {
	if instance, _ := get(child, "f").(string); strings.ContainsAny(instance, runtimeFlags) {
		return true
	}
	if s.ct == nil {
		return false
	}
	typ, _ := get(node, "t").(string)
	slot, err := s.ct.Declared(typ, child)
	if err != nil || slot == nil {
		return false
	}
	declared, _ := get(slot, "f").(string)
	return strings.ContainsAny(declared, runtimeFlags)
}

// msgRuntimeUnhashed is the plan note of a snapshot that holds runtime slots.
const msgRuntimeUnhashed = "runtime slots (transient, never persisted, such as an alarm extension's " +
	"timeInCurrentState) are marked runtime and shown without their values: they are not part of the " +
	"confirmation hash, so their ticking does not invalidate the token; the journal and n4_rollback keep them as " +
	"captured when the confirmed remove runs"

// hashedAdd is add, an inverse add op whose body the snapshot took, with the value of every
// runtime slot and of everything below it left out: nil when the body holds no runtime slot.
func (s *snapshot) hashedAdd(add *pyjson.Object) *pyjson.Object {
	if len(s.runtime) == 0 {
		return nil
	}
	body, _ := get(add, "b").(*pyjson.Object)
	if body == nil {
		return nil
	}
	hashed := s.unvalued(body, false)
	if !marked(hashed) {
		// every runtime slot the snapshot saw left the body (a link-driven one moves to prior):
		// the plan shows none, so it carries no runtime note (retro 2026-11-16 row M15b)
		return nil
	}
	out := copyObj(add)
	out.Set("b", hashed)
	return out
}

// marked says whether node or a node below it carries the "runtime" marker.
func marked(node *pyjson.Object) bool {
	if get(node, "runtime") == true {
		return true
	}
	kids, _ := get(node, "s").([]any)
	for _, k := range kids {
		if o, ok := k.(*pyjson.Object); ok && marked(o) {
			return true
		}
	}
	return false
}

// unvalued is a copy of node without the values of runtime slots (all values when below one).
func (s *snapshot) unvalued(node *pyjson.Object, below bool) *pyjson.Object {
	top := s.runtime[node] && !below
	below = below || s.runtime[node]
	out := pyjson.NewObject()
	for _, k := range node.Keys() {
		v, _ := node.Get(k)
		switch {
		case k == "v" && below:
			continue
		case k == "s":
			kids, _ := v.([]any)
			copied := make([]any, len(kids))
			for i, kid := range kids {
				if o, ok := kid.(*pyjson.Object); ok {
					copied[i] = s.unvalued(o, below)
				} else {
					copied[i] = kid
				}
			}
			v = copied
		}
		out.Set(k, v)
	}
	if top { // the marker sits on the runtime slot itself, never on its children
		out.Set("runtime", true)
	}
	return out
}
