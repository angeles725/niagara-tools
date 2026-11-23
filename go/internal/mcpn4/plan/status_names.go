package plan

import (
	"fmt"
	"strings"
)

// statusFlagNames are the flags of a baja:Status by bit, lowest first (box.StatusFlags names them
// the same way when a status is read).
var statusFlagNames = []string{"disabled", "fault", "down", "alarm", "stale", "overridden", "null", "unackedAlarm"}

// statusRule is the refusal of a status that is neither hex bits nor flag names.
var statusRule = "status must be a hex status string such as '0' or flag names (" + strings.Join(statusFlagNames, ", ") +
	", or ok) separated by commas such as 'fault' or 'down,overridden'"

// statusFromNames is the hex status string of a list of flag names ("fault", "down, overridden",
// "{alarm,unackedAlarm}", "ok" for none); false when a name is not a flag. n4_read_slots names the
// flags the same way, so a status read can be written back by name (retro 2026-11-15 row 5).
func statusFromNames(text string) (string, bool) {
	text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(text), "{"), "}"))
	bits := 0
	for _, name := range strings.Split(text, ",") {
		name = strings.TrimSpace(name)
		if name == "ok" {
			continue
		}
		found := false
		for i, n := range statusFlagNames {
			if n == name {
				bits |= 1 << i
				found = true
			}
		}
		if !found {
			return "", false
		}
	}
	return fmt.Sprintf("%x", bits), true
}
