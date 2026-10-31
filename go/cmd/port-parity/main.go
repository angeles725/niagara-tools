// Command port-parity diffs a reference frontend tree against its port and FAILs on every reference
// file, DOM id or named function the port lacks that the module's parity file does not waive with a
// recorded decision. Run it through toolbelt/lint-port-parity.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/portparity"
)

func main() { cli.Run(portparity.Main) }
