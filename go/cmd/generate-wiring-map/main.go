// Command generate-wiring-map scaffolds a facade's docs/wiring-map.md. Run it through
// toolbelt/generate-wiring-map.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/wiringmap"
)

func main() { cli.Run(wiringmap.Main) }
