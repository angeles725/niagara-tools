// Command schema-risk classifies the slot diff between two module snapshots (SAFE / LOSSY / OUTAGE). Run it
// through toolbelt/schema-risk.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/schemarisk"
)

func main() { cli.Run(schemarisk.Main) }
