// Command sweep-build-state validates BUILD-STATE.md + retros/INDEX.md (content check) or reports
// retro-debt aging (--age). Run it through toolbelt/sweep-build-state.sh, which builds this binary
// on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/buildstatefields"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(buildstatefields.Main) }
