// Command ci-steps prints every `steps:` item of a GitHub workflow as NUL-terminated
// <kind> <name> <body> records for scripts/ci-local.sh, which builds and runs it through
// go/go-tool.sh. `ci-steps changed` and `ci-steps lint-bats-heredoc` serve `ci-local.sh --changed`
// (go/internal/cisteps subcommands.go); `ci-steps bats-pin` reads the bats-core pin of the install step
// (batspin.go).
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cisteps"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

func main() { cli.Run(cisteps.Main) }
