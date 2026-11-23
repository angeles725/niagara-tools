// Command check-target-version compares the Niagara version a module jar was compiled against with the
// version of an installation it is about to be copied into.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/niagaraversion"
)

func main() { cli.Run(niagaraversion.Main) }
