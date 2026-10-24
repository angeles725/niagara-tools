// Command kit-state prints or writes the kit's generated state — build-n4-module-kit/BUILD-STATE.md
// (from BUILD-STATE.src.md + ledger.d/) and build-n4-module-kit/retros/INDEX.md (from the retro
// headers). Run it through toolbelt/kit-state.sh, which builds this binary on demand and passes the
// default kit root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/kitstate"
)

func main() { cli.Run(kitstate.Main) }
