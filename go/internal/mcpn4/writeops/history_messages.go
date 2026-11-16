package writeops

// The texts n4_add_history answers with, apart from the logic that picks them. Each %s is
// filled in the order its comment names.

const (
	// msgHistoryTypeMismatch: the ext_type, the point ORD, its family, its type, the fitting type.
	msgHistoryTypeMismatch = "ext_type %s does not fit %s, a %s point (%s): use %s or leave ext_type out"
	// msgHistoryNotAPoint: the ORD and its type.
	msgHistoryNotAPoint = "%s is a %s, not a numeric, boolean, enum or string control point: " +
		"a history extension goes under one of those"
	// msgHistoryNoRegistry: the ORD.
	msgHistoryNoRegistry = "the type of %s could not be read from the station's type registry (%s): " +
		"the extension type follows the point type"
	// msgHistoryIDTaken: the history id, the point ORD, the slot path of the extension that has it.
	msgHistoryIDTaken = "history id %s of %s is already used by the extension at %s: the new extension " +
		"would sit in fault \"Duplicate history id\"; use a history_name pattern that gives a new name, " +
		"or remove that extension first"
	// msgHistoryIDsUnread: the read error.
	msgHistoryIDsUnread = "could not read the station's history ids to check for a repeat (%s); nothing was planned"
	// msgHistoryNameRepeat: the history name and the two point ORDs.
	msgHistoryNameRepeat = "history name %s would repeat on %s and %s: use a history_name pattern that " +
		"includes the device, e.g. S%%parent.parent.parent.name%%_%%parent.name%%"

	noteHistoryRename    = "The station may rename on collision; the real inverse uses the assigned name."
	noteHistoryUnchecked = "history_name uses tokens other than %parent...name%: names were not checked for " +
		"repeats, in the batch or against the station's history ids."
	noteHistoryIDsCut = "The station's history id table was cut at the size or row cap: ids past the cut " +
		"were not checked for repeats."
	// noteHistoryDefaultCapacity: the duration and the interval, both readable.
	noteHistoryDefaultCapacity = "Capacity stays at the extension default (500 records, Roll): about %s at %s."
)
