// Command lint-vendor-floor parses every vendored browser library under <rc-dir>/vendor at the
// Chromium 83 panel floor and flags APIs above it. Run it through toolbelt/lint-vendor-floor.sh, which
// builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/vendorfloor"
)

func main() { cli.Run(vendorfloor.Main) }
