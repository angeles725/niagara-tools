// Command station-snapshot copies a station directory's config.bog and console*.txt into an out dir and writes
// manifest.json there. Run it through toolbelt/station-snapshot.sh, which builds this binary on demand and
// passes its own path (the prefix of the shell-style error messages).
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/stationsnapshot"
)

func main() { cli.Run(stationsnapshot.Main) }
