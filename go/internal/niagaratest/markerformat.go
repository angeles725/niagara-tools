package niagaratest

import "regexp"

// markerName is the marker file a run writes at the top of every scratch dir it creates or takes.
const markerName = "run-niagara-test.marker"

// markerHeader is the marker's first line, its format version.
const markerHeader = "run-niagara-test home marker 1"

// The marker's fields, in the order it writes them.
const (
	fieldRunID   = "run id"
	fieldInstall = "install"
	fieldSysSum  = "system.properties sha256"
	fieldLog     = "log"
)

// markerMax bounds the marker read; logHeadMax bounds the read of the kept run's log header.
const (
	markerMax  = 4 << 10
	logHeadMax = 64 << 10
)

// noSysProps is the marker's sha256 field for an install without defaults/system.properties.
const noSysProps = "absent"

// keptLogName is a run log's name as createLog makes it.
var keptLogName = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}Z-[A-Za-z0-9][A-Za-z0-9_.-]*\.log$`)
