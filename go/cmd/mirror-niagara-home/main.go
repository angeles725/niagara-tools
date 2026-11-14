// Command mirror-niagara-home builds a mirror of a Niagara install with a writable modules/. Run it through
// build-n4-module-kit/toolbelt/mirror-niagara-home.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mirrorhome"
)

func main() { cli.Run(mirrorhome.Main) }
