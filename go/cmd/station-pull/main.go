// Command station-pull copies a platform-daemon station with plat fget, verifies the CRCs and zips it
// (issue #1169). Run it through build-n4-module-kit/toolbelt/station-pull.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/stationpull"
)

func main() { cli.Run(stationpull.Main) }
