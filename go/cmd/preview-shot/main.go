// Command preview-shot takes a headless screenshot of a dashboard preview (issue #1126).
// Run it through build-n4-module-kit/toolbelt/preview-shot.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/previewshot"
)

func main() { cli.Run(previewshot.Main) }
