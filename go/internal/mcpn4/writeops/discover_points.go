package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// DiscoverPoints is n4_discover_points: the BACnet point discovery Workbench's Point Manager
// runs, submitted on the station. BBacnetPointDeviceExt.submitPointDiscoveryJob (hidden, no
// argument) submits a BacnetDiscoverPointsJob under Services/JobService that reads the device's
// objectList, then each object's name, primary value and description, and keeps one
// bacnet:DiscoveryPoint child (dc0..dcN) per object (lab proof: niagara-research
// odd/tasks/driver-discovery-remote-design-4.15.md section G). No point is added. The read-back
// lists the rows (discover_points_rows.go).
var DiscoverPoints = plan.Impl{ScopeOrds: func(args *pyjson.Object) []any { return []any{str(args, "device_ord")} },
	PlanWith: pointsPlan, Readback: pointsReadback}

const (
	pointsAction  = "submitPointDiscoveryJob"
	pointsDefWait = 120
	pointsMaxWait = 600
)

// pointsData is what the read-back needs: the points extension, the wait and the clock.
type pointsData struct {
	ext  string
	wait int
	w    *plan.State
}

func pointsPlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	wait, err := intArg(args, "wait_seconds", pointsDefWait, 1, pointsMaxWait)
	if err != nil {
		return nil, err
	}
	ext := str(args, "device_ord")
	h, nodes, err := plan.Handle(c, ext, 1)
	if err != nil {
		return nil, err
	}
	if !bacnetPointsExt(rootType(nodes)) {
		ext += "/points"
		if h, nodes, err = plan.Handle(c, ext, 1); err != nil {
			return nil, err
		}
	}
	if t := rootType(nodes); !bacnetPointsExt(t) {
		return nil, plan.ToolError(ext + " is " + pyjson.Repr(t) + ", not a BACnet points extension (give the device or its points)")
	}
	notes := []any{
		"reads the device's objectList, then each object's name, primary value and description (one ReadProperty or ReadPropertyMultiple group per object); waits up to " +
			strconv.Itoa(wait) + " s for the job",
		"no point is added to the station database: the rows are children of a job under Services/JobService",
		"the action is hidden and needs admin invoke rights on the points extension",
		"no inverse: field traffic cannot be undone",
	}
	return &plan.Planned{Ops: []any{invokeOp(obj("h", h, "a", pointsAction))}, Notes: notes,
		Data: pointsData{ext: ext, wait: wait, w: w}}, nil
}

// bacnetPointsExt is true for a BACnet points extension, the plain one or a vendor's subclass
// (honeywellBacnetSpyder:BacnetSpyderPointDeviceExt).
func bacnetPointsExt(t string) bool {
	return strings.HasSuffix(t, "PointDeviceExt") && strings.Contains(strings.ToLower(t), "bacnet")
}
