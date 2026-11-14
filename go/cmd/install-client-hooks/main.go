// Command install-client-hooks installs the kit's split-package pre-push gate into a client repository.
// Run it through scripts/install-client-hooks.sh, which builds this binary on demand; the script header is the contract.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/hookinstall"
)

func main() { cli.Run(hookinstall.ClientMain) }
