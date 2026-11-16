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
