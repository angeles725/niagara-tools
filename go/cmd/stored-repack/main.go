// Command stored-repack repackages a signed N4 module jar with every entry STORED. Run it through
// toolbelt/stored-repack.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/storedrepack"
)

func main() { cli.Run(storedrepack.Main) }
