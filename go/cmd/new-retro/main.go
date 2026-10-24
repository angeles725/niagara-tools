// Command new-retro writes a retro stub, regenerates retros/INDEX.md and flips retro_pending in
// BUILD-STATE.src.md. Run it through toolbelt/new-retro.sh, which builds this binary on demand and passes
// the default kit root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/newretro"
)

func main() { cli.Run(newretro.Main) }
