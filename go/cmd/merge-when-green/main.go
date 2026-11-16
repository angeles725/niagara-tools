// Command merge-when-green waits for the CI checks of a pull request's exact head commit and merges
// that commit (go/internal/mergewait). scripts/merge-when-green.sh builds and runs it.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mergewait"
)

func main() { cli.Run(mergewait.Main) }
