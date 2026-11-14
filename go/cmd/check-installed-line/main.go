// Command check-installed-line refuses a client branch that would downgrade a module group already
// installed. Run it through scripts/check-installed-line.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/checkinstalledline"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(checkinstalledline.Main) }
