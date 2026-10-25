// Command mcp-n4 is the Go port of the mcp-n4 stdio MCP server (mcp-n4-kit/docs/go-port-plan.md).
// It has been the default mcp-n4 server since slice S16 and the only one since S17 (the Python server is removed).
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	mcpcli "github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
)

func main() { cli.Run(mcpcli.Main) }
