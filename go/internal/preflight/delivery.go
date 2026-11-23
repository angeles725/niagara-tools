package preflight

import (
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// delivery prints what the pre-push gate will ask of a change to the kit when the gradle root lies in a kit
// checkout (a git work tree holding changelog.d/ and build-n4-module-kit/ledger.d/), so the fragments and the
// retro are written with the change and not found by the refusal. It is information, a PASS row: nothing here
// can fail a run. Silent elsewhere [ev: retro delivery-guards-and-ui-contrast Δ2].
func (r *run) delivery() {
	if r.win {
		return
	}
	out, ok := gitcmd.Run(r.gr, io.Discard, "rev-parse", "--show-toplevel")
	top := strings.TrimSpace(out)
	if !ok || top == "" || !lint.IsDir(top+"/changelog.d") || !lint.IsDir(top+"/build-n4-module-kit/ledger.d") {
		return
	}
	r.row("PASS", "delivery", top+" is a kit checkout: a change under go/, build-n4-module-kit/, tools/ or scripts/ "+
		"is pushed with a changelog.d/<date>-<slug>.md fragment (or a Changelog: none (<reason>) trailer), "+
		"a build-n4-module-kit/ledger.d/<date>-<slug>.md fragment committed alone and a pending retro "+
		"(toolbelt/new-retro.sh; or a Retro: none (trivial: <reason>) trailer): CONTRIBUTING.md 6.1 and 6.2")
}
