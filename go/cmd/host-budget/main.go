// Command host-budget reports and gates the shared-host resources (pids cgroups, the thread budget,
// the Go build cache, disk use) before heavy work. scripts/ci-local.sh runs it through
// go/go-tool.sh `go_tool_bin` before any CI step; see go/internal/hostbudget for the rules.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/hostbudget"
)

func main() { cli.Run(hostbudget.Main) }
