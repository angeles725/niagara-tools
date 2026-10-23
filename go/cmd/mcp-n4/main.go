// Command mcp-n4 is the Go port of the mcp-n4 stdio MCP server (mcp-n4-kit/docs/go-port-plan.md).
// It is opt-in until slice S16: the MCP config keeps launching the Python server.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
)

func main() { cli.Run(mcpcli.Main) }
