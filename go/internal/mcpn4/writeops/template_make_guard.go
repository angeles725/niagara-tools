package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The runaway guard of n4_make_station_template (issue #1057), confirmed live on a 4.14.0.162 station:
// BMakeStationTemplateJob zips the station home ^ (NewStationWideTemplateSource.getFilesToStore walks it and
// excludes only logs and the .ntpl files under stationTemplates), so it also zips ^temp, where it writes its
// own output. The .ntpl never reaches EOF: 66.9 GB after 95 min (about 12 MB/s) on a 68 KB config, 23 GB of host
// disk gone in under an hour. BSimpleJob.doCancel only interrupts the thread and the zip copy loop does not
// check the interrupt. No version check is trusted as evidence of a fixed build, so the dry run refuses unless
// the operator passes accept_runaway_risk=true.
const (
	msgMakeCause = "the station's MakeStationTemplateJob zips the whole station home ^, including ^temp, where it " +
		"writes its own output, so the .ntpl grows without end (confirmed on 4.14.0.162: 66.9 GB after 95 min, " +
		"about 12 MB/s, from a 68 KB config) and fills the host disk"
	msgMakeRecovery = "recovery: n4_save_station, stop the station process, delete the partial ^temp/*.ntpl, " +
		"restart the station; a job cancel does not stop it (it only interrupts the thread and the zip copy " +
		"does not check it)"
	msgMakeRefused = "n4_make_station_template refused: " + msgMakeCause + ". " + msgMakeRecovery +
		". Pass accept_runaway_risk=true only if you know this station's build does not zip ^temp into its " +
		"own output."
	msgMakeAccepted = "RUNAWAY RISK accepted (accept_runaway_risk=true): " + msgMakeCause + "; " + msgMakeRecovery
)

// makeRiskAccepted is the accept_runaway_risk argument (default false), or the tool error of a non-boolean.
func makeRiskAccepted(args *pyjson.Object) (bool, error) {
	raw, given := args.Get("accept_runaway_risk")
	if !given || raw == nil {
		return false, nil
	}
	b, ok := raw.(bool)
	if !ok {
		return false, plan.ToolError("accept_runaway_risk must be true or false, got " + pyjson.Repr(raw))
	}
	return b, nil
}
