// Command triage-console groups own-module exceptions (and, with --site, known site faults) of station console
// logs. Run it through toolbelt/triage-console.sh, which builds this binary on demand and passes the kit's
// default package, own-tag and module-prefix lists.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/triageconsole"
)

func main() { cli.Run(triageconsole.Main) }
