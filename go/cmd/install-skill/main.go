// Command install-skill installs a tracked skill launcher into an agent harness (Claude Code, pi, gentle-shell) and materializes the kit state.
// Run it through scripts/install-skill.sh, which builds this binary on demand; the script header is the contract.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/skillinstall"
)

func main() { cli.Run(skillinstall.Main) }
