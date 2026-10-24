// Command preflight is the BUILD-LOOP §0.b environment preflight. Run it through
// toolbelt/preflight.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/preflight"
)

func main() { cli.Run(preflight.Main) }
