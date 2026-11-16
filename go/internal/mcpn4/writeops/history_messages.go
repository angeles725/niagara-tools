package writeops

// The texts n4_add_history answers with, apart from the logic that picks them. Each %s is
// filled in the order its comment names.

const (
	// msgHistoryTypeMismatch: the ext_type, the point ORD, its family, its type, the fitting type.
	// The family carries its article ("a numeric", "an enum").
	msgHistoryTypeMismatch = "ext_type %s does not fit %s, %s point (%s): use %s or leave ext_type out"
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

	// msgHistoryModeType: the ext_type, its mode, the mode given.
	msgHistoryModeType    = "ext_type %s is a %s extension, but mode is %s"
	msgHistoryCovInterval = "interval_ms does not apply to mode cov: a change-of-value extension records " +
		"each change of the point's value and has no interval"
	msgHistoryNoInterval = "interval_ms is required for mode interval (300000 is 5 minutes)"
	// msgHistoryCapacity: the capacity given.
	msgHistoryCapacity = "capacity must be a whole number of records from 1 to 2147483647, or \"unlimited\", got %s"

	msgHistoryToleranceMode = "change_tolerance applies only to mode cov: an interval extension samples every " +
		"interval_ms and has no change tolerance"
	// msgHistoryToleranceType: the ext_type.
	msgHistoryToleranceType = "change_tolerance applies only to history:NumericCovHistoryExt, not ext_type %s"
	// msgHistoryTolerancePoint: the point ORD, its family, its type.
	msgHistoryTolerancePoint = "change_tolerance applies only to a numeric point's NumericCov extension: %s is " +
		"%s point (%s); add it in a separate call without change_tolerance"
	// msgHistoryTolerance: the change_tolerance given.
	msgHistoryTolerance = "change_tolerance must be a finite number of at least 0, in the point's units, got %s"

	noteHistoryRename    = "The station may rename on collision; the real inverse uses the assigned name."
	noteHistoryUnchecked = "history_name uses tokens other than %parent...name%: names were not checked for " +
		"repeats, in the batch or against the station's history ids."
	noteHistoryIDsCut = "The station's history id table was cut at the size or row cap: ids past the cut " +
		"were not checked for repeats."
	// noteHistoryDefaultCapacity: the full policy, the duration and the interval, both readable.
	noteHistoryDefaultCapacity = "Capacity stays at the extension default (500 records, %s): about %s at %s."
	// noteHistoryCapacity: the records, the full policy, the duration and the interval.
	noteHistoryCapacity = "Capacity %s records (%s): about %s at %s."
	// noteHistoryCovCapacity: the capacity text and the records.
	noteHistoryCovCapacity = "%s: a change-of-value history keeps the last %s changes; how long that lasts " +
		"depends on how often the value changes."
	noteHistoryCovDefault = "Capacity stays at the extension default (500 records, %s)"
	noteHistoryUnlimited  = "Capacity unlimited: the history keeps every record until the station's history " +
		"storage runs out."
	noteHistoryCapacityOp = "capacity is set by a second op per point on <ext_name>/historyConfig/capacity " +
		"(history:Capacity, 1:<records>, or 0:0 for unlimited): historyConfig is a component, which a station " +
		"refuses inside an add body."
	noteHistoryStopUnlimited = "full_policy stop has no effect with capacity unlimited: an unlimited history " +
		"is never full."
	noteHistoryFullPolicyOp = "full_policy is set by a set op per point on <ext_name>/historyConfig/fullPolicy " +
		"(history:FullPolicy): roll drops the oldest record when the capacity is reached, stop stops recording."
)
