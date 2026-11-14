// Command build-module builds a Niagara module: preflight, gradle on Java 8, the deployed-baseline drift
// gate, the verify gate and report-module. Run it through build-n4-module-kit/toolbelt/build.sh, which
// builds this binary on demand and names its own directory as the toolbelt.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/buildmodule"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(buildmodule.Main) }
