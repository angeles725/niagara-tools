// Command module-map adds a concern to a module (folder, pure class, test, MODULE-MAP row), says which concern
// owns a file and runs one concern's pure tests. Run it through toolbelt/module-map.sh, which builds this binary
// on demand and points MODULE_MAP_TOOLBELT at the toolbelt (for run-pure-test.sh).
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/modulemap"
)

func main() { cli.Run(modulemap.Main) }
