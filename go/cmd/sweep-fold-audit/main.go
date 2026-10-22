// Command sweep-fold-audit audits folded retros for [ev: retro] citation coverage. Run it through
// build-n4-module-kit/toolbelt/sweep-fold-audit.sh, which builds this binary on demand.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/foldaudit"
)

func main() { cli.Run(foldaudit.Main) }
