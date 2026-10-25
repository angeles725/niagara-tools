// Command run-niagara-test decides whether a Niagara test run passed from its captured output. Run it
// through toolbelt/run-niagara-test.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/niagaratest"
)

func main() { cli.Run(niagaratest.Main) }
