package blackbox

// The rig of the n4_register_bacnet_spyder tests: a verified writes-allowed server, a BACnet
// network under the write scope, a licence with room for one more device, and a station that
// answers what a Spyder does (its frozen slots, writeDeviceInstance, the instance question and the
// point discovery job).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	registerTool = "n4_register_bacnet_spyder"
	spyderType   = "honeywellBacnetSpyder:BacnetSpyder"
	spyNetOrd    = Folder + "/BacnetNetwork"
	spyJobOrd    = "slot:/Services/JobService/BacnetDiscoverPointsJob"
	instanceHdr  = "Slot Path,Object Id\n"
)

var spyArgs = Args{"network_ord": spyNetOrd, "name": "2016", "network_number": 2.0, "mac": 12.0, "instance": 2016.0}

// spyderRig is the rig plus what the fake controller does.
type spyderRig struct {
	*Rig
	sv        *boxstation.Servlets
	net       string // the network's handle
	inv       *invocations
	instances string // rows of the BQL over every BACnet device: slot path, object id
	online    string // the instance the controller reports
	hardware  string // the hardware id its discovery reads
	faulted   bool   // a new Spyder starts in fault
}

func newSpyderRig(t *testing.T, devices int) *spyderRig {
	t.Helper()
	s := &spyderRig{hardware: "33.0"}
	s.Rig, s.sv = licWriteRig(t, licCapacity(labLimits...), devicesCSV(devices))
	s.Station.SetSupertype(spyderType, "bacnet:BacnetDevice")
	s.Station.SetRegistry(map[string]any{spyderType: map[string]any{"p": "bacnet:BacnetDevice",
		"c": []any{dfxProp("config", nil), dfxProp("points", nil)}}}) // its frozen slots
	s.net = s.Add("3", "BacnetNetwork", "bacnet:BacnetNetwork").Handle
	s.inv = recordInvocations(s.Rig, nil)
	s.Station.Do(func(st *boxstation.Station) {
		st.AfterAdd = s.afterAdd
		st.Respond = s.respond
	})
	return s
}

// routes serves the capacity counts and the instance lookup.
func (s *spyderRig) routes(devices int) {
	routeOrds(s.sv, boxstation.OrdRoute{Needle: "from driver:Device", Body: []byte(devicesCSV(devices))},
		boxstation.OrdRoute{Needle: "from driver:ProxyExt", Body: []byte(usagePointsCSV)},
		boxstation.OrdRoute{Needle: "from bacnet:BacnetDevice", Body: []byte(instanceHdr + s.instances)})
}

// afterAdd gives a new Spyder its frozen slots, as the station does.
func (s *spyderRig) afterAdd(st *boxstation.Station, n *boxstation.Node) {
	if n.Type != spyderType {
		return
	}
	n.Children = append(n.Children, &boxstation.Node{Name: "address", Type: "bacnet:BacnetAddress"})
	st.Component(st.Component(n, "config", "bacnet:BacnetConfigDeviceExt", ""), "deviceObject", "bacnet:BacnetDeviceObject", "")
	st.Component(n, "points", "honeywellBacnetSpyder:BacnetSpyderPointDeviceExt", "")
	if s.faulted {
		n.Children = append(n.Children, &boxstation.Node{Name: "status", Type: "baja:Status", Value: "2"},
			&boxstation.Node{Name: "faultCause", Type: "baja:String", Value: "Device limit exceeded"})
	}
}

func (s *spyderRig) respond(st *boxstation.Station, action string, n *boxstation.Node) (any, bool) {
	switch action {
	case "writeDeviceInstance":
		id := n.Child("config").Child("deviceObject").Child("objectId")
		s.online = strings.TrimPrefix(id.Value.(string), "device:")
		return nil, true
	case "getOnlineDeviceInstanceNumber":
		return map[string]any{"t": "baja:Integer", "v": s.online}, true
	case "submitPointDiscoveryJob":
		jobs := st.Component(comServices(st), "JobService", "baja:Folder", "")
		job := st.Component(jobs, "BacnetDiscoverPointsJob", "bacnet:BacnetDiscoverPointsJob", "")
		job.Children = append(job.Children, &boxstation.Node{Name: "jobState", Type: "baja:JobState", Value: "success"})
		for i, row := range [][3]string{{"AV_HardwareID", "analogValue:0", s.hardware}, {"RA_T", "analogInput:1", "21.5"}} {
			dc := st.Component(job, "dc"+string(rune('0'+i)), "bacnet:DiscoveryPoint", "")
			for _, kv := range [][3]string{{"objectName", "baja:String", row[0]}, {"objectId", "bacnet:BacnetObjectIdentifier", row[1]},
				{"value", "baja:Double", row[2]}, {"description", "baja:String", ""}} {
				dc.Children = append(dc.Children, &boxstation.Node{Name: kv[0], Type: kv[1], Value: kv[2]})
			}
		}
		return map[string]any{"t": "baja:Ord", "v": spyJobOrd}, true
	}
	return nil, false
}

// device is the handle of the Spyder the tool created ("" when there is none).
func (s *spyderRig) device(name string) (h string) {
	s.Station.Do(func(st *boxstation.Station) {
		if n := st.ByHandle(s.net).Child(name); n != nil {
			h = n.Handle
		}
	})
	return h
}

func stepsOf(t *testing.T, plan map[string]any) []map[string]any {
	t.Helper()
	return objects(Dig(plan, "plan", "follow_up"))
}

// deviceChild is the handle of the child named name of the node with handle h.
func (s *spyderRig) deviceChild(h, name string) (out string) {
	s.Station.Do(func(st *boxstation.Station) { out = st.ByHandle(h).Child(name).Handle })
	return out
}
