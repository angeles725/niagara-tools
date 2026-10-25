// Command report-module is the aggregated conformance report of a Niagara N4 module. Run it through
// build-n4-module-kit/toolbelt/report-module.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/reportmodule"
)

func main() { cli.Run(reportmodule.Main) }
