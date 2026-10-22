// Command gen-build-state generates the kit block's last_change, open_issues items and session log in
// build-n4-module-kit/BUILD-STATE.md from build-n4-module-kit/ledger.d/. Run it through
// toolbelt/gen-build-state.sh, which builds this binary on demand and passes the default kit root.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/buildstate"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(buildstate.Main) }
