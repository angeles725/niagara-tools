// Package rules is the list of every lint rule n4lint knows. Adding a rule: write its package under
// rules/<name> exporting Spec() lint.Spec, then add that Spec() call to All.
package rules

import (
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/bqlconcat"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/clockzerofloor"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/configsanity"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/delays"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/demandscope"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/driverscaffold"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/extordvalue"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/extwritable"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/nomd5credential"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/nosystemout"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/nullcontext"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/sedisplay"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/servicetypes"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/setnullord"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/silentprotection"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/subscribeleak"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/timers"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/wbfilechooser"
)

// All returns every registered rule, in the order --all runs them.
func All() []lint.Spec {
	return []lint.Spec{
		nullcontext.Spec(),
		timers.Spec(),
		driverscaffold.Spec(),
		setnullord.Spec(),
		extordvalue.Spec(),
		servicetypes.Spec(),
		silentprotection.Spec(),
		nosystemout.Spec(),
		sedisplay.Spec(),
		bqlconcat.Spec(),
		clockzerofloor.Spec(),
		subscribeleak.Spec(),
		wbfilechooser.Spec(),
		nomd5credential.Spec(),
		configsanity.Spec(),
		delays.Spec(),
		demandscope.Spec(),
		extwritable.Spec(),
	}
}
