package writeops

// flagLetter is one slot flag: its letter in a flags string and its name in plans and notes.
type flagLetter struct{ letter, name string }

// flagTable is every slot flag of a Niagara station, in the order the station encodes a flags
// string (bajaScript rc/baja/comp/Flags.js Flags.flags, javax.baja.sys.Flags). It is data only:
// the copy notes and n4_set_flags name flags from it.
var flagTable = []flagLetter{
	{"r", "readonly"},
	{"t", "transient"},
	{"h", "hidden"},
	{"s", "summary"},
	{"a", "async"},
	{"n", "no-run"},
	{"d", "default-on-clone"},
	{"c", "confirm-required"},
	{"o", "operator"},
	{"x", "execute-on-change"},
	{"f", "fan-in"},
	{"A", "no-audit"},
	{"p", "composite"},
	{"R", "remove-on-clone"},
	{"m", "metadata"},
	{"N", "non-critical"},
	{"L", "link-target"},
	{"1", "user-defined-1"},
	{"2", "user-defined-2"},
	{"3", "user-defined-3"},
	{"4", "user-defined-4"},
}
