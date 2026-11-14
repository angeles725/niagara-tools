// Command live-smoke is the live smoke runner of the mcp-n4 server (formerly
// mcp-n4-kit/tools/live_smoke.py): plan only by default; with --apply it starts the sibling
// mcp-n4 binary with --allow-writes over stdio and runs the kitControl thermostat scenario,
// every write as dry run -> confirmation token -> execute -> verdict.
//
//	live-smoke [--apply --station NAME=URL --station-home NAME=PATH] [--credential-env PREFIX]
//	           [--insecure-tls NAME] [--state-dir DIR] [--report PATH] [--settle SECONDS]
//	           [--call-timeout SECONDS] [--allow-http-for-tests] [--go-bin PATH]
//
// The server is --go-bin, else $MCP_N4_GO_BIN, else mcp-n4 next to this binary. Credential
// values (<PREFIX>_USER, <PREFIX>_PASSWORD) are scrubbed from all output. Exit: 0 plan printed
// or every required step verified; 1 a run that did not verify; 2 usage.
package main

import (
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/livesmoke"
)

func main() {
	cli.Run(func(args []string, s cli.Stdio) int { return livesmoke.Main(args, s, os.Environ()) })
}
