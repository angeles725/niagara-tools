// Command n4call is the bench client of the mcp-n4 server: it starts the server command given
// after "--" over stdio and runs the tool calls of a JSON-lines file in one session, a write as
// dry run -> confirmation token -> execute. One JSON line of output per request; credential
// values are masked.
//
//	n4call --calls FILE [--dry-run-only] [--call-timeout SECONDS] -- SERVER [SERVER ARGS...]
//
// Exit: 0 every call answered without error; 1 a tool error or a server failure; 2 usage.
package main

import (
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/benchcall"
)

func main() {
	cli.Run(func(args []string, s cli.Stdio) int { return benchcall.Main(args, s, os.Environ()) })
}
