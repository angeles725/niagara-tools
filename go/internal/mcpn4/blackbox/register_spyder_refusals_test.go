package blackbox

// The refusals of n4_register_bacnet_spyder that come before a token: nothing is created, nothing
// is journaled and no token is issued.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// networkChildren is how many children the BACnet network holds.
func (s *spyderRig) networkChildren() (n int) {
	s.Station.Do(func(st *boxstation.Station) { n = len(st.ByHandle(s.net).Children) })
	return n
}

// refusedRegister is a dry run refused with every text in want, leaving no journal line and no
// device under the network.
func refusedRegister(t *testing.T, s *spyderRig, args Args, want ...string) {
	t.Helper()
	before := s.networkChildren()
	refusedDry(t, s.Rig, registerTool, args, want...)
	if after := s.networkChildren(); after != before {
		t.Errorf("a refused registration changed the network's children from %d to %d", before, after)
	}
	if len(s.inv.get()) != 0 {
		t.Errorf("a refused registration invoked %v", s.inv.get())
	}
}

func TestRegisterSpyderIsRefusedOverTheLicenceDeviceLimit(t *testing.T) {
	s := newSpyderRig(t, 26)
	refusedRegister(t, s, spyArgs, "device.limit 26", "already counts 26 device(s)")
}

func TestRegisterSpyderIsRefusedForANameInUseWithTheEscapedLookupHint(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.Add(s.net, "$32016", spyderType)
	refusedRegister(t, s, spyArgs, "$32016", "already in use", "name = '$2432016'")
}

func TestRegisterSpyderIsRefusedForAnInstanceInUse(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.instances = "slot:/Folder/Other/Vav7,device:2016\nslot:/Folder/Other/Vav8,device:2017\n"
	s.routes(5)
	refusedRegister(t, s, spyArgs, "device:2016", "already used by slot:/Folder/Other/Vav7")
}

func TestRegisterSpyderIsNeverPlannedAtTheFactoryInstance(t *testing.T) {
	s := newSpyderRig(t, 5)
	refusedRegister(t, s, withArgs(spyArgs, Args{"instance": 9999.0}), "9999", "factory instance")
}

func TestRegisterSpyderIsRefusedOnADisabledOrFaultedNetwork(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.Set(s.net, "status", map[string]any{"nm": "p", "t": "baja:Status", "v": "1"})
	refusedRegister(t, s, spyArgs, spyNetOrd, "disabled")
	s = newSpyderRig(t, 5)
	s.Set(s.net, "status", map[string]any{"nm": "p", "t": "baja:Status", "v": "2"})
	s.Set(s.net, "faultCause", map[string]any{"nm": "p", "t": "baja:String", "v": "Port not found"})
	refusedRegister(t, s, spyArgs, spyNetOrd, "in fault", "Port not found")
}

func TestRegisterSpyderNeedsABacnetNetworkAndValidAddresses(t *testing.T) {
	s := newSpyderRig(t, 5)
	refusedRegister(t, s, withArgs(spyArgs, Args{"network_ord": Folder}), "not a bacnet:BacnetNetwork")
	refusedRegister(t, s, withArgs(spyArgs, Args{"name": "a/b"}), "plain slot name")
	for _, bad := range []Args{{"mac": 255.0}, {"mac": -1.0}, {"network_number": 0.0}, {"instance": 4194303.0}} {
		if code := s.Srv.CallError(registerTool, withArgs(spyArgs, bad)); code != -32602 {
			t.Errorf("%v: error code %d", bad, code)
		}
	}
}

func TestRegisterSpyderModelCheckNeedsDiscoveryAndAKnownModel(t *testing.T) {
	s := newSpyderRig(t, 5)
	refusedRegister(t, s, withArgs(spyArgs, Args{"expected_model": "PUB6438S", "discover": false}), "expected_model", "discover")
	refusedRegister(t, s, withArgs(spyArgs, Args{"expected_model": "XYZ"}), "XYZ", "PUB1012S")
}

// writeDeviceInstance has no inverse and a duplicate instance makes the station rebind addresses
// (B1165): an instance lookup that cannot be read is a refusal before the token, not a note.
func TestRegisterSpyderIsRefusedWhenTheInstanceLookupCannotBeRead(t *testing.T) {
	const why = "refusing before the token"
	s := newSpyderRig(t, 5)
	routeOrds(s.sv, boxstation.OrdRoute{Needle: "from driver:Device", Body: []byte(devicesCSV(5))},
		boxstation.OrdRoute{Needle: "from driver:ProxyExt", Body: []byte(usagePointsCSV)}) // no BQL route: the station answers 404
	refusedRegister(t, s, spyArgs, "could not be read", why, "HTTP 404")
	s.instances = strings.Repeat("slot:/Folder/Other/Vav,device:1\n", 50001)
	s.routes(5)
	refusedRegister(t, s, spyArgs, "could not be read", why, "more than 50000 rows")
	s = newSpyderRig(t, 5)
	routeOrds(s.sv, boxstation.OrdRoute{Needle: "from driver:Device", Body: []byte(devicesCSV(5))},
		boxstation.OrdRoute{Needle: "from driver:ProxyExt", Body: []byte(usagePointsCSV)},
		boxstation.OrdRoute{Needle: "from bacnet:BacnetDevice", Body: []byte("Slot Path\nslot:/Folder/Other/Vav7\n")})
	refusedRegister(t, s, spyArgs, "could not be read", why, "two columns")
}
