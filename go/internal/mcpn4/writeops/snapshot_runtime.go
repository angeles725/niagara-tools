package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A runtime slot is one whose value the station computes or never persists, so it can change
// between a dry run and its execute by timing alone (retro 2026-11-16 row M15: an alarm
// extension's timeInCurrentState, BAlarmSourceExt flags 3 = readonly|transient, ticks every
// second). It is a frozen slot whose DECLARED flags hold r readonly or t transient
// (computedFlags, the rule keepsOutput already applies to an out), or any slot whose instance
// flags hold t (never persisted). Such a slot stays in the snapshot the journal keeps and a
// rollback re-adds, as before; only its value, and the values below it, leave the plan the
// token hashes. Its name and type stay, so a slot or child added, removed or retyped still
// changes the hash, and so does a configuration value.

// runtimeSlot says whether child, loaded under node, is a runtime slot. A contract that cannot be
// read leaves only the instance rule: a slot it cannot classify stays hashed (the safe side).
func (s *snapshot) runtimeSlot(node, child *pyjson.Object) bool {
	if instance, _ := get(child, "f").(string); strings.Contains(instance, "t") {
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
	return strings.ContainsAny(declared, computedFlags)
}

// msgRuntimeUnhashed is the plan note of a snapshot that holds runtime slots.
const msgRuntimeUnhashed = "runtime slots (declared readonly or transient, such as an alarm extension's " +
	"timeInCurrentState) are shown without their values and are not part of the confirmation hash, " +
	"so their ticking does not invalidate the token; the journal and n4_rollback keep them as " +
	"captured when the confirmed remove runs"

// hashedAdd is add, an inverse add op whose body the snapshot took, with the value of every
// runtime slot and of everything below it left out: nil when the body holds no runtime slot.
func (s *snapshot) hashedAdd(add *pyjson.Object) *pyjson.Object {
	if len(s.runtime) == 0 {
		return nil
	}
	body, _ := get(add, "b").(*pyjson.Object)
	out := copyObj(add)
	out.Set("b", s.unvalued(body, false))
	return out
}

// unvalued is a copy of node without the values of runtime slots (all values when below one).
func (s *snapshot) unvalued(node *pyjson.Object, below bool) *pyjson.Object {
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
	return out
}
