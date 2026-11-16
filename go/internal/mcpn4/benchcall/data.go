// Package benchcall is the n4call command: it starts an mcp-n4 server over stdio and runs a
// list of tool calls against it, one JSON line of output per call. It is the bench client for
// live probes: the operator chooses the server command line (stations, scopes, tier opt-ins),
// n4call adds nothing to it.
package benchcall

const prog = "n4call"

const usageText = `usage: n4call --calls FILE [--execute [--allow-destructive] | --dry-run-only] [--call-timeout SECONDS] -- SERVER [SERVER ARGS...]

Runs every line of FILE against the mcp-n4 server started as SERVER, in order, in one session.
A line is {"tool": NAME, "args": {...}} for a read, with "write": true for a write tool: the
write is sent with dry_run=true and, only when --execute is given, executed with the
confirmation_token that dry run returned. Without --execute every write is a dry run only
(--dry-run-only says so explicitly and excludes --execute), so a lost flag never executes.
A destructive write (the server's destructiveHint: remove, rollback, save, ack_alarm, set_tags)
executes only when --execute and --allow-destructive are given AND its own line carries
"execute": true; otherwise a run with --execute is refused before the server starts. A run with
writes says on stderr whether they execute. Output: one JSON line per request,
{"tool", "phase": read|dry_run|execute, "error", "result"}. Credential values named by the
server's --credential-env are masked. The run stops at the first tool error.
Exit: 0 every call answered without error; 1 a tool error or a server failure; 2 usage.
`

// defaultCallTimeout bounds one request, in seconds.
const defaultCallTimeout = 120.0

// Phases of an output line.
const (
	phaseRead    = "read"
	phaseDryRun  = "dry_run"
	phaseExecute = "execute"
)
