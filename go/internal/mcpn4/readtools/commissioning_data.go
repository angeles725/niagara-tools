package readtools

import "regexp"

// What n4_commissioning_report reads, bounds and says (data only).

// The report's sections, in reply order.
var commSections = []string{"points", "histories", "alarms", "links", "naming", "station"}

// Defaults of the report (the registry schema bounds the arguments).
const (
	commDefaultLimit     = 20
	commDefaultLinkDepth = 2
	commDefaultTimeout   = 60
	// alarmScanDepth is how many component levels below the AlarmService the recipient links
	// are scanned at (classes and recipients sit right below it, or one folder down).
	alarmScanDepth = 2
	// servicesDepth loads the services, the services inside a container, and their slots.
	servicesDepth = 3
)

// The fixed places the report reads besides its scope.
const (
	alarmServiceOrd = "station:|slot:/Services/AlarmService"
	servicesOrd     = "station:|slot:/Services"
)

// commQuery is one table of the report: its select, the base it runs over ("" is the scope)
// and its columns by position (the headers are display names a station translates).
type commQuery struct {
	query, base string
	names       []string
}

// The tables. status.bits is the status as a number, which no locale translates; the device
// and network selects are the health tool's.
var commQueries = map[string]commQuery{
	"networks": {healthQueries[0].query, "", []string{"Slot Path", "Name", "Type", bitsColumn, enabledColumn}},
	"devices":  {healthQueries[1].query, "", []string{"Slot Path", "Name", "Type", bitsColumn, enabledColumn}},
	"points": {"select slotPath, name, displayName, type, status.bits from control:ControlPoint", "",
		[]string{"Slot Path", "Name", displayColumn, "Type", bitsColumn}},
	"histories": {"select slotPath, name, type, enabled, status.bits, historyConfig.id, lastRecord.timestamp from history:HistoryExt", "",
		[]string{"Slot Path", "Name", "Type", enabledColumn, bitsColumn, "Id", "Last"}},
	"alarms": {"select slotPath, name, type, alarmClass from alarm:AlarmSourceExt", "",
		[]string{"Slot Path", "Name", "Type", "Class"}},
	"classes": {"select slotPath, name, openAlarmCount, unackedAlarmCount from alarm:AlarmClass", alarmServiceOrd,
		[]string{"Slot Path", "Name", "Open", "Unacked"}},
}

const displayColumn = "Display"

// pointFlagChecks maps a status flag to the points check that counts it, in reply order; a
// flag missing here (unackedAlarm) is not a commissioning finding of its own.
var pointFlagChecks = [][2]string{{"fault", "in_fault"}, {"down", "down"}, {"stale", "stale"}, {"disabled", "disabled"},
	{"overridden", "overridden"}, {"alarm", "in_alarm"}, {nullFlag, "null_output"}, {unknownFlag, "status_unreadable"}}

// deviceListedFlags are the flags that list a device (or a network) itself: its points are then
// counted, not listed.
var deviceListedFlags = map[string]bool{"down": true, "disabled": true, "fault": true}

// faultFlag is the flag of a history extension or a service in fault.
const faultFlag = "fault"

// nullCell is how ITableToCsv writes a null value (an AbsTime that was never set).
const nullCell = "null"

// expectKinds maps a point kind of an expect filter to the text its type name holds.
var expectKinds = map[string]string{"numeric": "Numeric", "boolean": "Boolean", "enum": "Enum", "string": "String",
	"writable": "Writable"}

// paletteDefaultNames are the names of the points of the stock control palette (its Points
// folder), a default with or without the digits the station appends. The simple name of a
// point's own type is a default only with those digits (paletteDefaultFact).
var paletteDefaultNames = []string{"NumericPoint", "NumericWritable", "BooleanPoint", "BooleanWritable", "EnumPoint",
	"EnumWritable", "StringPoint", "StringWritable"}

var (
	// escapedNameRe is an escape of a slot name: $xx or $uxxxx.
	escapedNameRe = regexp.MustCompile(`\$(?:u[0-9A-Fa-f]{4}|[0-9A-Fa-f]{2})`)
	// trailingDigitsRe is the number a station appends to a repeated name.
	trailingDigitsRe = regexp.MustCompile(`[0-9]+$`)
	// priorityInputRe is an input slot of a writable point: in1 to in16, one per priority level.
	priorityInputRe = regexp.MustCompile(`^in(?:[1-9]|1[0-6])$`)
)

// Messages of the report.
const (
	commBadScope    = "scope must be a plain slot path such as station:|slot:/Drivers (default: the whole station), got %s"
	commBadSection  = "sections must be a list of %s; got %s"
	commBadExpect   = "%s must be an object with kinds (a list of %s) and/or name_pattern (a glob such as *Temp*, matched against the point's name and display name): %s"
	commTruncated   = "truncated at %d components: counts are lower bounds and absences are not asserted; narrow the scope"
	commTruncShort  = " (truncated at %d components, narrow the scope)"
	commQueryFailed = "the %s query failed: %s"
	commCutAbsence  = "the %s query was truncated at %d components: an absence cannot be asserted; narrow the scope"
	commNoAnswer    = "the station answered no %s for any of the %d row(s)"
	commRequests    = "station reads made: one GET per query or page, one slot load per tree (a load may poll more than once)"

	sumOK         = "%s: ok"
	sumFindings   = "%s: %d finding(s)"
	sumAlsoNot    = ", %d check(s) not checked"
	sumPartial    = "%s: no findings in the checks that ran, %d check(s) not checked"
	sumNotChecked = "%s: not checked: %s"
	sumSkipped    = "%s: skipped: %s"

	factStatus        = "status %s"
	factDevice        = "%s; %d point(s) below it, %d not ok"
	factNoHistory     = "no history extension below it"
	factHistDisabled  = "enabled false (history id %s)"
	factHistStatus    = "status %s (history id %s)"
	factNeverRecorded = "lastRecord.timestamp is null (history id %s)"
	factNoAlarm       = "no alarm extension below it"
	factUnknownClass  = "alarmClass %s names no alarm class of the station"
	factNoRecipient   = "no link from its alarm topic to a component within %d levels of the AlarmService"
	factOpen          = "%d open alarm(s)"
	factUnacked       = "%d unacknowledged alarm(s)"
	factSourceMissing = "link %s into %s: source slot path %s does not exist"
	factSourceForm    = "link %s into %s: source ORD %s is neither a handle nor an absolute slot path"
	factSeveralLinks  = "slot %s is the target of %d links: %s"
	factDangling      = "out is read by no link seen (type %s)"
	factDuplicate     = "display name %q is shared by %d siblings"
	factEscaped       = "slot name %s holds escaped characters (%s)"
	factPaletteType   = "still called %s, the name of its own type %s and the number the station appends to a repeated name"
	factPaletteList   = "still called by the control palette default name %s"
	noteExpectHistory = "pass expect_histories to list them"
	factService       = "status %s"

	skipNoExpectAlarms = "expect_alarms not given: which points should have an alarm extension is the caller's to say"
	skipNotStation     = "the scope is not the whole station"
	whyUnresolved      = "%d link(s) below the AlarmService have an unresolved or unseen source: a class not proven linked may be linked by one of them"
	whyLinkCap         = "the link scan of %s at depth %d loaded %d components, over the cap of %d: narrow the scope or lower link_depth"
	whyUnsaved         = "the station exposes no unsaved-changes state to a slot read or a query; n4_save_station saves and proves persistence"
	whyBackup          = "the station exposes no time of its last backup to a slot read or a query; check the backup files on the supervisor or the engineering PC"
	whyNoClock         = "the nearest expiry cannot be compared with the station date: %s"
	noteLinks          = "Links are read %d component level(s) below the scope (link_depth). Whether a link's source or target slot exists is not checked: actions and topics are not part of a slot load. A source by handle that is not among the loaded components is outside the scan or gone (source_outside_scan), never judged."
	noteNaming         = "Names are those of the control points in scope; folders and devices are not read."
	noteClassScope     = "Classes are those the alarm extensions in scope name; open and unacknowledged counts are the class's own, for the whole station."
)
