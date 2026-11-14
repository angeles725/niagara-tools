// Command check-bats-skips fails when a bats TAP holds a skip whose reason matches no line of the
// allowlist. Run it through scripts/check-bats-skips.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/checkbatsskips"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(checkbatsskips.Main) }
