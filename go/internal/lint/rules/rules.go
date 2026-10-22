// Package rules is the list of every lint rule n4lint knows. Adding a rule: write its package under
// rules/<name> exporting Spec() lint.Spec, then add that Spec() call to All.
package rules

import (
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/driverscaffold"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/nullcontext"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/servicetypes"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/timers"
)

// All returns every registered rule, in the order --all runs them.
func All() []lint.Spec {
	return []lint.Spec{
		nullcontext.Spec(),
		timers.Spec(),
		driverscaffold.Spec(),
		servicetypes.Spec(),
	}
}
