// Package kitticket is kit-ticket (toolbelt/kit-ticket.sh). RED stub: the port is not written yet.
package kitticket

import "github.com/angeles725/niagara-tools/go/internal/cli"

// DefaultRootEnv and ScriptEnv are set by the toolbelt wrapper.
const (
	DefaultRootEnv = "KIT_TICKET_DEFAULT_ROOT"
	ScriptEnv      = "KIT_TICKET_SCRIPT"
)

// Host is what kit-ticket reads from outside its arguments.
type Host struct {
	Today func() string
	Pid   int
	Env   []string
}

// Command is kit-ticket over h.
func Command(h Host) cli.Main { return func([]string, cli.Stdio) int { return 99 } }
