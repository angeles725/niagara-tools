// Command bog-audit audits a station config.bog against the deployed module source. Run it through
// build-n4-module-kit/toolbelt/bog-audit.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/bogaudit"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(bogaudit.Main) }
