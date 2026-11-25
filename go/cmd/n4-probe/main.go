// Command n4-probe is the kit's live probe of a station's HTTP API: `stall` times GETs of one URL,
// `pulse` presses a point and times how its read-back fields change and return. It is rate-capped
// (4 Hz), has one observer and a fixed run length, and reads credentials only from the operator env
// file. The MCP server's own stdio driver is go/cmd/mcp-n4/n4call. Build it with
// `cd go && go build -o bin/n4-probe ./cmd/n4-probe`.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/n4probe"
)

func main() { cli.Run(n4probe.Main) }
