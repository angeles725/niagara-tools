// Command check-skill-drift reports whether the installed skill launcher and prompt file of each harness still match the tracked ones (read-only).
// Run it through scripts/check-skill-drift.sh, which builds this binary on demand; the script header is the contract.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/skillinstall"
)

func main() { cli.Run(skillinstall.DriftMain) }
