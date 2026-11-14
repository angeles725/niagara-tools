package vendorfloor

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// Every text the lint prints, apart from the logic that picks it.
const (
	tool  = "lint-vendor-floor" // the prefix of a stderr line
	rowID = "vendor-floor"      // the check column of every row

	usage = "usage: lint-vendor-floor.sh <rc-dir> [--strict]"

	noVendorNote = tool + ": no vendor/ under %s — nothing to check\n"
	noFilesNote  = tool + ": no *.js under %s/vendor — nothing to check\n"
	envFaultLine = tool + ": env fault: %s\n"

	nodeMissing  = "unavailable: node not found (install Node, or set KIT_NODE)"
	acornMissing = "unavailable: acorn not resolvable from '%s' (npm install --prefix toolbelt/eslint, or set KIT_ACORN)"

	syntaxFail = "syntax: does not parse at ecmaVersion %d (Chromium %d floor): %s"
	moduleOnly = "syntax: parses only as an ES module (%s) — load with type=\"module\" or vendor the UMD/global build"
	apiAbove   = "api: %s needs Chrome %d (floor %d) — feature-guarded? record it in THIRD-PARTY.md"
)

// skipRow is the row of a run that could not check anything: "SKIP  vendor-floor  <rc-dir>  <reason>".
func skipRow(rc, reason string) string {
	return lint.Finding{Rule: rowID, Status: "SKIP", Msg: rc + "  " + reason}.Row()
}

// apiMsg words one use of an API above the floor; call is the use as written (".at(", "new WeakRef").
func apiMsg(call string, chrome int) string {
	return fmt.Sprintf(apiAbove, call, chrome, FloorChrome)
}
