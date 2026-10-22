// Command ci-steps prints every `steps:` item of a GitHub workflow as NUL-terminated
// <kind> <name> <body> records for scripts/ci-local.sh, which builds and runs it through
// go/go-tool.sh.
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cisteps"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(cisteps.Main) }
