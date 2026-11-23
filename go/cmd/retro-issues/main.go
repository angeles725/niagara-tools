// Command retro-issues turns the mcp-n4 / build-n4-module deltas of retros into GitHub issues (dry run by default).
// Run it through build-n4-module-kit/toolbelt/retro-issues.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retroissues"
)

func main() { cli.Run(retroissues.Main) }
