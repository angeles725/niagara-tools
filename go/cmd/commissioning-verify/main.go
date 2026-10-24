// Command commissioning-verify is the §6.b commissioning punch-list auditor. Run it through
// build-n4-module-kit/toolbelt/commissioning-verify.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/commissioningverify"
)

func main() { cli.Run(commissioningverify.Main) }
