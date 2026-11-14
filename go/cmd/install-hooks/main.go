// Command install-hooks activates this repository's .githooks (core.hooksPath) or deactivates them again.
// Run it through scripts/install-hooks.sh, which builds this binary on demand; the script header is the contract.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/hookinstall"
)

func main() { cli.Run(hookinstall.Main) }
