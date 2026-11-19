package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A Status value (baja:StatusBoolean, StatusNumeric, StatusEnum, StatusString) declares its status
// child readonly, yet the status is configuration: a schedule whose in and defaultOutput held {null}
// was copied with {ok} on every copy, and each copy's out became a live false / 0.0 (delta P2, live
// near-miss 2026-11-19). A copy carries such a status whole with its value, as n4_set_slot writes
// a Status slot (plan.ToBSON), whenever it is off its declared default; at the default the copy's
// type supplies it and it stays out, as before. The read-back strips the copy alike, so a status
// the station did not keep is a difference.

// statusSimple is the type a carried status child is written as (plan.ToBSON's status child).
const statusSimple = "baja:Status"

// carriedStatus says whether kid, a child of a node of type parentType declared with default def,
// is the status of a Status value held off its default: carried instead of left out.
func carriedStatus(parentType string, kid, def *pyjson.Object) bool {
	return parentType != statusSimple && statusType(parentType) && name(kid) == "status" &&
		!sameValue(kid, def)
}

// asStatusChild is a carried status child typed as plan.ToBSON writes it.
func asStatusChild(kid *pyjson.Object) *pyjson.Object {
	if get(kid, "t") == nil {
		kid.Set("t", statusSimple)
	}
	return kid
}
