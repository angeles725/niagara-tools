// Command plat-read runs one read-only plat verb against a Niagara platform daemon (issue #850).
// Run it through build-n4-module-kit/toolbelt/plat-read.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/platread"
)

func main() { cli.Run(platread.Main) }
