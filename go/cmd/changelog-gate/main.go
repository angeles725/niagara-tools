// Command changelog-gate fails a range that changes kit behaviour (go/, build-n4-module-kit/, tools/)
// without a CHANGELOG entry. Run it through scripts/changelog-gate.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/changelog"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(changelog.Main) }
