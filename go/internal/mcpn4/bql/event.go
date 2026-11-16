package bql

import "strings"

// Alarm events (retro 2026-11-16 P3): the alarm database keeps one record per alarm event,
// created with a new uuid when the source leaves normal and updated in place with normalTime
// when it returns and with ackTime and user when it is acknowledged (alarm-rt AlarmSupport
// newAlarm and ToNormalTransition.run). The event query reads each record's times as machine
// text, so a localized station still answers ISO-8601 with its offset.

// AlarmEventQuery reads one row per event, by position: alarm time, uuid, source, source and
// ack state, class, transition, normal time, ack time and the acknowledging user.
const AlarmEventQuery = "select timestamp.encodeToString, uuid, source, sourceState.encodeToString, " +
	"ackState.encodeToString, alarmClass, alarmTransition.encodeToString, normalTime.encodeToString, " +
	"ackTime.encodeToString, user"

// RecordSourcePath is the slot path a record's source names (local:|station:|slot:/a/b,
// station:|slot:/a/b or slot:/a/b), each segment a slot name; ok is false for any other source
// (several ORDs, another scheme, an empty or root path).
func RecordSourcePath(source string) (string, bool) {
	cut := strings.LastIndex(source, "slot:")
	if cut < 0 {
		return "", false
	}
	prefix, path := source[:cut], source[cut+len("slot:"):]
	switch prefix {
	case "", "station:|", "local:|station:|":
	default:
		return "", false
	}
	if !strings.HasPrefix(path, "/") || path == "/" {
		return "", false
	}
	for _, seg := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if !slotNameRe.MatchString(seg) {
			return "", false
		}
	}
	return path, true
}
