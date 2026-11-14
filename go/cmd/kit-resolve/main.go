// Command kit-resolve prints the build-n4-module kit a session may run, or refuses a stale, dirty or diverged one.
// Run it through scripts/kit-resolve.sh, which builds this binary on demand; the script header is the contract.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/kitresolve"
)

func main() { cli.Run(kitresolve.Main) }
