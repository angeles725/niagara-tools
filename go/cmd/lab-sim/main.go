// Command lab-sim is a field-device simulator for the laboratory: a Modbus TCP
// server and BACnet/IP devices driven by a scenario file, with a request log
// and a loopback control port (go/internal/labsim/README.md).
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim"
)

func main() { cli.Run(labsim.Main) }
