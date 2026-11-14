// Command check-client-source reports whether a module root is the checkout its BUILD-STATE declares as
// the source of truth. Run it through scripts/check-client-source.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/checkclientsource"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(checkclientsource.Main) }
