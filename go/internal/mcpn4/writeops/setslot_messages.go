package writeops

// The refusal of n4_set_slot for a slot the load does not list (lab retro 2026-11-16 row M8).

// msgSetSlotMissing is a slot neither the load lists nor any type contract the station describes
// declares: the slot (repr) and the component's ORD. The station leaves out a frozen property whose
// value equals the default its type DECLARES (a 10 s RelTime, false, a non-zero number), not only
// 0, false or an empty string; plan.Resolve fills such a slot from the type registry, so this
// refusal is left for a misspelled name, a dynamic slot or a registry the station did not answer.
const msgSetSlotMissing = "slot %s not found on %s. The station leaves out a frozen property whose " +
	"value equals its declared default (whatever default its type declares, e.g. 10 s or false, not " +
	"only 0, false or an empty string), and n4_set_slot fills such a slot from the type registry; " +
	"no type contract the station describes declares this one, so n4_set_slot cannot read its type " +
	"and refuses. Check the name with n4_read_slots; if it is a dynamic slot at its default, set it " +
	"once in Workbench (or through an action such as `set`) and plan again"

// #990 M3: a templateConfig slot bound to a component attribute or to a target the plan cannot place.
const (
	msgConfigAttribute = "refusing to set templateConfig slot %s: its ConfigBinding targets %s of %s, so the " +
		"station would rename that component; rename it with n4_rename_component instead"
	msgConfigTarget = "refusing: the ConfigBinding target %s is not a handle found in the template instance %s " +
		"(read %d levels down), so the slot the station would write cannot be placed in the write scope"
	msgConfigScope = "refusing: the station propagates this templateConfig slot to %s slot %s, which is outside " +
		"the write scope: %s"
)
