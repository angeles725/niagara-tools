// Package stationsnapshot is station-snapshot (toolbelt/station-snapshot.sh). RED stub: the port follows.
package stationsnapshot

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// ScriptEnv names the script path the bash shell put in front of its own error messages.
const ScriptEnv = "STATION_SNAPSHOT_SCRIPT"

// Host is what station-snapshot reads besides its arguments: the clock.
type Host struct{ Now func() time.Time }

// Command is station-snapshot over h.
func Command(h Host) cli.Main { return func(args []string, s cli.Stdio) int { return 0 } }
