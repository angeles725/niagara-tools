// Command obix-link-audit diffs a wiring map's source column against the links into one component (a
// saved oBIX response or one read-only oBIX GET). Run it through toolbelt/obix-link-audit.sh, which
// builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/obixlinkaudit"
)

func main() { cli.Run(obixlinkaudit.Main) }
