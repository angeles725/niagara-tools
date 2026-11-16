// Command mcp-n4 is the Go port of the mcp-n4 stdio MCP server (mcp-n4-kit/docs/go-port-plan.md).
// It has been the default mcp-n4 server since slice S16 and the only one since S17 (the Python server is removed).
//
// Two subcommands set it up for agent harnesses: `mcp-n4 install` copies this binary to a stable
// path, `mcp-n4 register` writes a station's launcher and harness entry. The server itself takes
// only options (a positional is a usage error there), so a first argument naming a subcommand is
// never a server command line.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/install"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/register"
)

func run(args []string, s cli.Stdio) int {
	if len(args) > 0 {
		switch args[0] {
		case "install":
			return install.Main(args[1:], s)
		case "register":
			return register.Main(args[1:], s)
		}
	}
	return mcpcli.Main(args, s)
}

func main() { cli.Run(run) }
