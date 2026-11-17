// Command mstp-plan plans BACnet MS/TP MACs set by DIP switches: MAC, switches ON and device ID per
// device, refusing reserved, out-of-range or already-used addresses.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mstpplan"
)

func main() { cli.Run(mstpplan.Main) }
