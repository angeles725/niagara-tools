// Command lab-station starts, stops and restarts a local lab station without an interactive console (issue #848).
// Run it through build-n4-module-kit/toolbelt/lab-station.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labstation"
)

func main() { cli.Run(labstation.Main) }
