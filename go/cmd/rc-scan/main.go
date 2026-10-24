// Command rc-scan is the browser-resource lint of a module's rc/ web assets. Run it through
// toolbelt/rc-scan.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/rcscan"
)

func main() { cli.Run(rcscan.Main) }
