package commissioningverify

import (
	"bytes"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/bogaudit"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stationCheck is one bog-audit check folded into the punch-list: its CHECK id and its row check name. Its FAIL,
// WARN and PASS rows are relayed (the bash fold dropped the WARN rows of CHECK13 and CHECK15-20, so the WARN-only
// CHECK15 and CHECK19 never reached the punch-list).
type stationCheck struct{ id, check string }

var stationChecks = []stationCheck{
	{"XML", "bog-xml"}, // what the bog holds that Niagara reads and XML 1.0 forbids (xmlsrc.Doc.Warnings)
	{"CHECK11", "proxy-link-safety"},
	{"CHECK13", "station-logic:relay-double-source"},
	{"CHECK14", "station-logic:own-output-unlinked"},
	{"CHECK15", "station-logic:sensor-crossed"},
	{"CHECK16", "station-logic:defrost-sibling"},
	{"CHECK17", "station-logic:room-index-mismatch"},
	{"CHECK18", "station-logic:tile-number"},
	{"CHECK19", "station-logic:link-direction"},
	{"CHECK20", "numeric-to-bool-direct"},
}

// stationChecks runs bog-audit in process on the bog and relays its XML, CHECK11 and CHECK13-20 rows, check by check
// in that order; a bog-audit exit 3 is one SKIP row.
func (r *report) stationChecks(o options) {
	args := []string{o.bog, "--module", o.mod}
	if o.strict {
		args = append(args, "--strict")
	}
	var b bytes.Buffer
	if bogaudit.Main(args, cli.Stdio{Out: &b, Err: &b}) == 3 {
		r.emit(skip, "station-checks", "bog-audit exit 3 (unreadable or malformed bog, or no component of the module) — run bog-audit.sh manually")
		return
	}
	lines := strings.Split(b.String(), "\n")
	for _, c := range stationChecks {
		for _, ln := range lines {
			for _, st := range []string{failed, warn, pass} {
				if p := c.id + "  " + st; strings.HasPrefix(ln, p) {
					r.emit(st, c.check, strings.TrimPrefix(ln, p+"  "))
				}
			}
		}
	}
}
