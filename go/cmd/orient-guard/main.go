// Command orient-guard is the BUILD-LOOP §0.a location guard. Run it through
// build-n4-module-kit/toolbelt/orient-guard.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/orientguard"
)

func main() { cli.Run(orientguard.Main) }
