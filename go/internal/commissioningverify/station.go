package commissioningverify

import (
	"bytes"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/bogaudit"
	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stationCheck is one bog-audit check folded into the punch-list: its CHECK id, its row check name and the
// statuses relayed (bog-audit's other rows of that id are dropped).
type stationCheck struct {
	id, check string
	statuses  []string
}

var stationChecks = []stationCheck{
	{"CHECK11", "proxy-link-safety", []string{failed, pass, warn}},
	{"CHECK13", "station-logic:relay-double-source", []string{failed, pass}},
	{"CHECK14", "station-logic:own-output-unlinked", []string{failed, warn, pass}},
	{"CHECK15", "station-logic:sensor-crossed", []string{failed, pass}},
	{"CHECK16", "station-logic:defrost-sibling", []string{failed, pass}},
	{"CHECK17", "station-logic:room-index-mismatch", []string{failed, pass}},
	{"CHECK18", "station-logic:tile-number", []string{failed, pass}},
	{"CHECK19", "station-logic:link-direction", []string{failed, pass}},
	{"CHECK20", "numeric-to-bool-direct", []string{failed, pass}},
}

// stationChecks runs bog-audit in process on the bog and relays its CHECK11 and CHECK13-20 rows, check by check
// in that order; a bog-audit exit 3 is one SKIP row.
func (r *report) stationChecks(o options) {
	args := []string{o.bog, "--module", o.mod}
	if o.strict {
		args = append(args, "--strict")
	}
	var b bytes.Buffer
	if bogaudit.Main(args, cli.Stdio{Out: &b, Err: &b}) == 3 {
		r.emit(skip, "station-checks", "bog-audit env fault (exit 3) — check python3 availability and bog path")
		return
	}
	lines := strings.Split(b.String(), "\n")
	for _, c := range stationChecks {
		for _, ln := range lines {
			for _, st := range c.statuses {
				if p := c.id + "  " + st; strings.HasPrefix(ln, p) {
					r.emit(st, c.check, strings.TrimPrefix(ln, p+"  "))
				}
			}
		}
	}
}
