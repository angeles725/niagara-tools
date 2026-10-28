// Command ledger-check checks a client's VERSION-LEDGER.md against the kit's templates/VERSION-LEDGER.md. Run it
// through toolbelt/ledger-check.sh, which builds this binary on demand and points LEDGER_CHECK_TEMPLATE at the
// kit's template.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/ledgercheck"
)

func main() { cli.Run(ledgercheck.Main) }
