package writeops

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// The texts of n4_set_flags: its plan notes and its refusals (data; the logic is in setflags.go
// and setflags_current.go).

// flagsUnprovenNote is the probe-only declaration METHODOLOGY section 2 asks of a flags write.
const flagsUnprovenNote = "setting slot flags is not yet proven on a live station: try it on a scratch slot first"

// Why a slot's current flags cannot be read from its type contract.
const (
	flagsNoRegistry = "this client cannot read the station's type definitions (registry channel)"
	flagsNoType     = "the component has no type"
	flagsNotAString = "the station sent flags that are not a string"
)

func flagsChangeNote(slot, ord, current, requested string) string {
	return "flags of slot " + pyjson.StrRepr(slot) + " on " + ord + ": current " + shownFlags(current) +
		" -> requested " + shownFlags(requested)
}

func flagsNoSlotMsg(slot, ord, typ string) string {
	return "slot " + pyjson.StrRepr(slot) + " does not exist on " + ord + ": the station does not list it and its type " +
		typ + " does not declare it; check the name with n4_read_slots"
}

func flagsNothingToDoMsg(slot, ord, current string) string {
	return "slot " + pyjson.StrRepr(slot) + " on " + ord + " already has flags " + shownFlags(current) + ": nothing to do"
}

func flagsUnknownMsg(slot, ord, why string) string {
	return "cannot determine the current flags of slot " + pyjson.StrRepr(slot) + " on " + ord +
		": the station sends no flags for it and " + why + "; it may be a frozen slot at the flags its type " +
		"declares, and without them no faithful inverse exists, so nothing is planned"
}

func flagsMalformedMsg(slot, ord string) string {
	return "cannot determine the current flags of slot " + pyjson.StrRepr(slot) + " on " + ord + ": " + flagsNotAString
}

func flagsChangedSinceMsg(slot any, ord string, batch any, wrote, now string) string {
	return "the flags of slot " + pyjson.Repr(slot) + " on " + ord + " changed since batch " + pyjson.Str(batch) +
		" set them (the batch set " + shownFlags(wrote) + ", the station now holds " + shownFlags(now) +
		"): refusing to overwrite a later change. Restore them by hand with n4_set_flags if that is still wanted"
}
