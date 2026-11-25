package subscribeleakjs

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule; tests/fixtures/lint-subscribe-js holds the same
// sources for tests/lint-subscribe-js.bats). sjs-leak and sjs-strict: a Subscriber never unsubscribed, WARN and FAIL
// under --strict; sjs-paired: the tutorial finally, a widget doDestroy hook and a single unsubscribe each pair it;
// sjs-detach-only: detach() alone does not; sjs-notcode: a comment, a string or a local function named
// unsubscribeAll does not pair it, a commented or quoted `new baja.Subscriber()` is no site; sjs-subscribe: a
// .subscribe( on a Subscriber-named receiver (once per file, and only when the file has no `new` site);
// sjs-mixin: a file that loads bajaux/mixin/subscriberMixIn (its destroy calls unsubscribeAll) pairs its sites, a
// comment naming it does not; sjs-mixin-code: only a module id in a define/require list or an import counts, never a
// block comment, an unrelated string, another call's argument or a longer id; sjs-waiver-subscribe: a waiver on the first
// .subscribe( row (a later line's waiver does not help); sjs-bare-paired: a bare marker FAILs in a file with a teardown or the mixin; sjs-two-sites: one row per site; sjs-waiver, sjs-waiver-bare and sjs-waiver-js: the waiver comment;
// sjs-skip: *.min.js, build/, node_modules/ and Java are not read; sjs-usage, sjs-bad-flag, sjs-nondir and
// sjs-empty-root: the argument and root errors.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 19) }
