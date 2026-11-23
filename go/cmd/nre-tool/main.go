// Command nre-tool runs a Niagara install's build-time nre tools by name (issue #851).
// Run it through build-n4-module-kit/toolbelt/nre-tool.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/nretool"
)

func main() { cli.Run(nretool.Main) }
