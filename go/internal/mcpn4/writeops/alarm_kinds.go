package writeops

// alarmKind is one palette alarm extension: every one is the type alarm:AlarmSourceExt with
// preset offnormalAlgorithm and faultAlgorithm children (alarm-rt module.palette, folder
// Extensions). family is the point kind its algorithm accepts (each algorithm's
// isGrandparentLegal: BOutOfRangeAlgorithm -> BNumericPoint, BBooleanChangeOfStateAlgorithm ->
// BBooleanPoint, ...; "" is any control point, BAlarmSourceExt.isParentLegal), fault "" keeps
// the extension's default alarm:FaultAlgorithm, params are the algorithm arguments it takes.
type alarmKind struct {
	name, offnormal, fault, family, palette string
	params                                  []string
}

// alarmExtType is the type of every palette alarm extension.
const alarmExtType = "alarm:AlarmSourceExt"

// alarmKinds is the kind table, in the order refusals list them.
var alarmKinds = []alarmKind{
	{"out_of_range", "alarm:OutOfRangeAlgorithm", "alarm:OutOfRangeFaultAlgorithm", "Numeric", "OutOfRangeAlarmExt",
		[]string{"high_limit", "low_limit", "deadband", "high_limit_text", "low_limit_text"}},
	{"numeric_change_of_state", "alarm:NumericChangeOfStateAlgorithm", "", "Numeric", "NumericChangeOfStateAlarmExt",
		[]string{"alarm_values"}},
	{"boolean_change_of_state", "alarm:BooleanChangeOfStateAlgorithm", "", "Boolean", "BooleanChangeOfStateAlarmExt",
		[]string{"alarm_value"}},
	{"boolean_command_failure", "alarm:BooleanCommandFailureAlgorithm", "", "Boolean", "BooleanCommandFailureAlarmExt", nil},
	{"enum_change_of_state", "alarm:EnumChangeOfStateAlgorithm", "alarm:EnumFaultAlgorithm", "Enum", "EnumChangeOfStateAlarmExt",
		[]string{"alarm_values"}},
	{"enum_command_failure", "alarm:EnumCommandFailureAlgorithm", "alarm:EnumFaultAlgorithm", "Enum", "EnumCommandFailureAlarmExt", nil},
	{"string_change_of_value", "alarm:StringChangeOfStateAlgorithm", "alarm:StringChangeOfStateFaultAlgorithm", "String",
		"StringChangeOfValueAlarmExt", nil},
	{"status", "alarm:StatusAlgorithm", "alarm:StatusFaultAlgorithm", "", "StatusAlarmExt", nil},
}

// alarmDefaultKind is the kind a point of each family gets when kind is not given.
var alarmDefaultKind = map[string]string{"Numeric": "out_of_range", "Boolean": "boolean_change_of_state",
	"Enum": "enum_change_of_state", "String": "string_change_of_value"}

// alarmAlgorithmParams are every algorithm argument, in the order the ops set them.
var alarmAlgorithmParams = []string{"high_limit", "low_limit", "deadband", "high_limit_text", "low_limit_text",
	"alarm_value", "alarm_values"}

func findAlarmKind(name string) (alarmKind, bool) {
	for _, k := range alarmKinds {
		if k.name == name {
			return k, true
		}
	}
	return alarmKind{}, false
}

// fits is whether the kind's algorithms accept a point of family.
func (k alarmKind) fits(family string) bool { return k.family == "" || k.family == family }

// takes is whether param is one of the kind's algorithm arguments.
func (k alarmKind) takes(param string) bool {
	for _, p := range k.params {
		if p == param {
			return true
		}
	}
	return false
}

// alarmKindNames are the kind names, in table order.
func alarmKindNames() []string {
	out := []string{}
	for _, k := range alarmKinds {
		out = append(out, k.name)
	}
	return out
}

// fittingAlarmKinds are the kinds that fit a point of family.
func fittingAlarmKinds(family string) []string {
	out := []string{}
	for _, k := range alarmKinds {
		if k.fits(family) {
			out = append(out, k.name)
		}
	}
	return out
}

// alarmTransitionBits encodes alarm_enable: BAlarmTransitionBits.encodeToString is the bit mask
// in hex (Integer.toHexString), TO_OFFNORMAL 0x1, TO_FAULT 0x2, TO_NORMAL 0x4, TO_ALERT 0x8.
var alarmTransitionBits = map[string]int{"offnormal": 1, "fault": 2, "normal": 4, "alert": 8}
