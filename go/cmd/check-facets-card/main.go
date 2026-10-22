// Command check-facets-card diffs the BFacets key reference card against the BFacets key
// constants. Run it through toolbelt/check-facets-card.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/facetscard"
)

func main() { cli.Run(facetscard.Main) }
