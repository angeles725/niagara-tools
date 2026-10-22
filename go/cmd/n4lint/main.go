// Command n4lint runs the kit's Java lint rules (internal/lint, rules in internal/lint/rules).
// Each toolbelt/lint-<name>.sh wrapper builds it on demand and runs "n4lint lint-<name> <args>...".
package main

import (
	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules"
)

var command = lint.Command(rules.All()...)

func main() { cli.Run(command) }
