// Command gen-retro-index generates build-n4-module-kit/retros/INDEX.md from each
// retro's header comments. Run it through toolbelt/gen-retro-index.sh, which builds
// this binary on demand and passes the default kit root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/retroindex"
)

func main() { cli.Run(retroindex.Main) }
