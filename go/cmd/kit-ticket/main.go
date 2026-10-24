// Command kit-ticket opens a kit defect issue with gh, or writes it to retros/tickets/ when gh is absent or
// fails. Run it through toolbelt/kit-ticket.sh, which builds this binary on demand and passes the default kit
// root and its own path.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/kitticket"
)

func main() { cli.Run(kitticket.Main) }
