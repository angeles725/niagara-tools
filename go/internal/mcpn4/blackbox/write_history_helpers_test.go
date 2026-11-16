package blackbox

// Helpers of the n4_add_history tests: a writes-allowed station that also answers BQL, knows the
// control point supertypes, and gives each new history extension its frozen historyConfig with
// the id the station assigns ("/<station>/<parent name>" for the pattern %parent.name%).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// noHistoryIDs is the history id table of a station without history extensions.
const noHistoryIDs = "Slot Path,Id\n"

var pointSupertypes = [][2]string{{"control:NumericWritable", "control:NumericPoint"},
	{"control:BooleanWritable", "control:BooleanPoint"}, {"control:NumericPoint", "baja:Component"},
	{"control:BooleanPoint", "baja:Component"}}

// historyRig is a verified writes-allowed rig whose station serves the history id table ids.
// faultCause, when not "", is the fault every new extension sits in.
type historyRig struct {
	*Rig
	sv         *boxstation.Servlets
	faultCause string
}

func newHistoryRig(t *testing.T, ids string) *historyRig {
	t.Helper()
	h := &historyRig{}
	h.Rig = NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		h.sv = boxstation.ServeServlets(t, r.Station)
		o.Stations = map[string]string{StationName: h.sv.URL}
	})
	h.ConnectVerified()
	for _, p := range pointSupertypes {
		h.Station.SetSupertype(p[0], p[1])
	}
	routeOrds(h.sv, boxstation.OrdRoute{Needle: "history:HistoryExt", Body: []byte(ids)})
	h.Station.Do(func(st *boxstation.Station) { st.AfterAdd = h.assignID })
	return h
}

// assignID is what the station does when a history extension starts: its historyConfig gets the
// id built from the station name and the parent's name, and its fault cause is set when asked.
func (h *historyRig) assignID(st *boxstation.Station, node *boxstation.Node) {
	if !strings.HasPrefix(node.Type, "history:") {
		return
	}
	parent := parentOf(st.Root(), node)
	config := st.Component(node, "historyConfig", "history:HistoryConfig", "")
	config.Children = append(config.Children, &boxstation.Node{Name: "id", Type: "history:HistoryId",
		Value: "/" + StationName + "/" + parent.Name})
	if h.faultCause != "" {
		node.Children = append(node.Children, &boxstation.Node{Name: "faultCause", Type: "baja:String",
			Value: h.faultCause})
	}
}

func parentOf(n, child *boxstation.Node) *boxstation.Node {
	for _, c := range n.Children {
		if c == child {
			return n
		}
		if p := parentOf(c, child); p != nil {
			return p
		}
	}
	return nil
}

// historyOf is the type of the child name of the component with handle h, "" when absent.
func historyOf(r *Rig, h, name string) (typ string) {
	r.Station.Do(func(st *boxstation.Station) {
		if c := st.ByHandle(h).Child(name); c != nil {
			typ = c.Type
		}
	})
	return typ
}

// historyArgs are n4_add_history arguments for points, %parent.name% names, 5 min, plus extra.
func historyArgs(extra Args, points ...string) Args {
	list := []any{}
	for _, p := range points {
		list = append(list, Folder+"/"+p)
	}
	return withArgs(Args{"points": list, "history_name": "%parent.name%", "interval_ms": 300000}, extra)
}
