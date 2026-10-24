// Command kit-cite-resolve resolves every evidence pointer of /build-n4-module kit cards ([ev: …] tags and
// prose citations) and prints its status and an excerpt, so a reviewer only judges whether the source
// says it. Run it through build-n4-module-kit/toolbelt/kit-cite-resolve.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/citeresolve"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(citeresolve.Main) }
