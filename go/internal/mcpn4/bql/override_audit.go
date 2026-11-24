package bql

import (
	"strconv"
	"strings"
)

// The station reads of n4_override_audit (retro R8 delta N7). Live on a lab station
// (4.15): `select slotPath, out, overrideExpiration from control:NumericWritable` works, a
// `where overrideExpiration != null` returns no row, and the active level shows only inside the
// out text ("0,0 {disabled,stale} @ 16"), so every writable is read and the level is parsed.

// WritableTypes are the writable point types read, one query each (BNumericWritable,
// BBooleanWritable, BEnumWritable, BStringWritable).
var WritableTypes = []string{"control:NumericWritable", "control:BooleanWritable", "control:EnumWritable", "control:StringWritable"}

// OverrideQuery reads, by position, a writable's slot path, its out as the station shows it and
// its overrideExpiration (the text "null" when no override expires).
func OverrideQuery(typ string) string {
	return "select slotPath, out, overrideExpiration from " + typ
}

// ScheduleHandlesQuery reads each schedule's slot path and its handle (BComponent.getHandle, a
// public getter BQL lists as a field), the key a link's sourceOrd ("h:<handle>") names it by.
const ScheduleHandlesQuery = "select slotPath, handle from schedule:ControlSchedule"

// Level names of an out text that is not a priority level.
const (
	LevelFallback = "def"
	LevelUnknown  = "unknown"
)

// OutLevel is the active level an out text ends with: "75.0 {ok} @ 8" gives 8, "5 {ok} @ def"
// the fallback (LevelFallback), anything else LevelUnknown. level is 0 unless text names one of
// the 16 inputs.
func OutLevel(text string) (level int, name string) {
	i := strings.LastIndex(text, "@")
	if i < 0 {
		return 0, LevelUnknown
	}
	tail := strings.TrimSpace(text[i+1:])
	if strings.EqualFold(tail, LevelFallback) {
		return 0, LevelFallback
	}
	if n, err := strconv.Atoi(tail); err == nil && n >= 1 && n <= 16 {
		return n, tail
	}
	return 0, LevelUnknown
}
