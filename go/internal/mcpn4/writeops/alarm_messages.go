package writeops

// The texts n4_add_alarm answers with, apart from the logic that picks them. Each %s is filled
// in the order its comment names.

const (
	// msgAlarmKind: the kind given, the kind names.
	msgAlarmKind = "kind must be one of %s, got %s"
	// msgAlarmKindMismatch: the kind, the point ORD, its family, its type, the fitting kinds.
	msgAlarmKindMismatch = "kind %s does not fit %s, a %s point (%s): use one of %s, or leave kind out"
	// msgAlarmNotAPoint: the ORD and its type.
	msgAlarmNotAPoint = "%s is a %s, not a numeric, boolean, enum or string control point: an alarm " +
		"extension goes under one of those"
	// msgAlarmNoRegistry: the ORD and the error.
	msgAlarmNoRegistry = "the type of %s could not be read from the station's type registry (%s): " +
		"the alarm kind follows the point type"
	// msgAlarmParam: the argument, the kind, the point ORD, the kinds that take it.
	msgAlarmParam = "%s does not apply to kind %s of %s: it is an argument of %s"
	// msgAlarmSecond: the point ORD, the kind, the existing extension's name.
	msgAlarmSecond = "%s already has a %s alarm extension, %s: configure that one with n4_set_slot, or pass " +
		"allow_second=true to add another"
	// msgAlarmChild: the point ORD and the name.
	msgAlarmChild = "%s already has a child named %s: remove it first or pass another ext_name"
	// msgAlarmClass: the class given, the existing class names.
	msgAlarmClass = "alarm class %s does not exist on the station; existing classes: %s. An alarm class " +
		"lives under the AlarmService (or an alarm:AlarmClassFolder) and alarm_class is its slot name"
	// msgAlarmClassesUnread: the read error.
	msgAlarmClassesUnread = "could not read the station's alarm classes to check alarm_class (%s); nothing was planned"
	// msgAlarmEnable: the value given.
	msgAlarmEnable = "alarm_enable must be a list of offnormal, fault, normal and alert, got %s"
	// msgAlarmValues: the value given.
	msgAlarmValues = "alarm_values must be a non-empty object of tag -> ordinal (whole number), " +
		"e.g. {\"alarm\": 1}, got %s"
	// msgAlarmArg: the argument and the coercion error.
	msgAlarmArg = "%s: %s"
	// msgAlarmString: the argument and the value given.
	msgAlarmString     = "%s must be a non-empty string, got %s"
	msgAlarmDeadband   = "deadband must be 0 or more (its slot facet min is 0)"
	msgAlarmNoPoints   = "points must list 1 to %d point ORDs"
	msgAlarmPointOrd   = "points must be distinct non-empty ORD strings, got %s"
	noteAlarmRename    = "The station may rename on collision; the real inverse uses the assigned name."
	noteAlarmAlgorithm = "Each extension is added with its scalar slots, then its algorithms are set by type " +
		"(offnormalAlgorithm, faultAlgorithm) and their slots by path, in the same batch: an algorithm is a " +
		"component, which a station refuses inside an add body."
	noteAlarmNoLimits = "kind out_of_range without high_limit or low_limit: both limits stay disabled, the " +
		"extension raises no offnormal alarm until a limit is enabled."
	noteAlarmClassesCut = "The station's alarm class table was cut at the size or row cap and alarm_class is not " +
		"among the classes before the cut: it was not checked."
)
