// Command lint-guard-pins checks that every toolbelt/lint-*.sh declares a `# Mutation:` guard-pin
// that resolves to a tests/*.bats fixture. Run it through toolbelt/lint-guard-pins.sh, which builds
// this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/guardpins"
)

func main() { cli.Run(guardpins.Main) }
