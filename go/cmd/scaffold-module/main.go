// Command scaffold-module emits a buildable N4 module skeleton from a bundled fixture. Run it through
// build-n4-module-kit/toolbelt/scaffold-module.sh, which builds this binary on demand and tells it which
// toolbelt it was invoked from.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/scaffoldmodule"
)

func main() { cli.Run(scaffoldmodule.Main) }
