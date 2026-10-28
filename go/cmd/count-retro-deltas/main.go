// Command count-retro-deltas counts each retro's Proposed kit deltas rows and compares them with its Delta count,
// its deltas header and its retros/INDEX.md row. Run it through toolbelt/count-retro-deltas.sh, which builds this
// binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retrodeltas"
)

func main() { cli.Run(retrodeltas.Main) }
