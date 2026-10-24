// Command slot-coverage is the MM2 exposed-set coverage of a module (set-coverage, the type-set parse
// mode and per-slot). Run it through toolbelt/slot-coverage.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/slotcoverage"
)

func main() { cli.Run(slotcoverage.Main) }
