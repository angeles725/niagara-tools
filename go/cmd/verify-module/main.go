// Command verify-module is THE gate for built N4 module jars (and its coverage and --plano modes). Run it
// through toolbelt/verify-module.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/verifymodule"
)

func main() { cli.Run(verifymodule.Main) }
