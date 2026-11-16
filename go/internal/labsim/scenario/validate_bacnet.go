package scenario

import (
	"net/netip"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
)

// pointType reads an object type a point can be: any but the Device.
func pointType(name string) (bacnet.ObjectType, bacobj.Kind, bool) {
	t, ok := bacnet.ParseObjectType(name)
	if !ok {
		return 0, 0, false
	}
	kind, ok := bacobj.KindOf(t)
	return t, kind, ok
}

func isAnalog(objectType string) bool {
	_, kind, ok := pointType(objectType)
	return ok && kind == bacobj.Analog
}

// pointTypeNames lists the object types a point can be.
func pointTypeNames() string {
	var names []string
	for _, name := range bacnet.ObjectTypeNames() {
		if _, _, ok := pointType(name); ok {
			names = append(names, name)
		}
	}
	return strings.Join(names, ", ")
}

// bacnetSeen is what must be unique across the devices of a scenario.
type bacnetSeen struct {
	instances map[int]string
	listens   map[string]string
}

// validateBACnet checks the BACnet section of a device and the mappings of
// its points.
func validateBACnet(d *Device, where string, seen *bacnetSeen, out *faults) {
	if b := d.BACnet; b != nil {
		validateBACnetDevice(d, where+": bacnet", seen, out)
	}
	objects := map[bacnet.ObjectID]string{}
	for i := range d.Points {
		p := &d.Points[i]
		if p.BACnet == nil {
			continue
		}
		at := where + ", " + label("point", i, p.Name) + ": bacnet"
		if d.BACnet == nil {
			out.add(at, msgBACnetNoDevice)
			continue
		}
		validateBACnetPoint(p, at, objects, out)
	}
}

func validateBACnetDevice(d *Device, where string, seen *bacnetSeen, out *faults) {
	b := d.BACnet
	switch {
	case b.DeviceInstance == nil:
		out.add(where, msgNoDeviceInstance)
	case *b.DeviceInstance < 0 || *b.DeviceInstance > MaxInstance:
		out.add(where, msgDeviceInstance, *b.DeviceInstance, MaxInstance)
	case seen.instances[*b.DeviceInstance] != "":
		out.add(where, msgDeviceInstanceUsed, *b.DeviceInstance, seen.instances[*b.DeviceInstance])
	default:
		seen.instances[*b.DeviceInstance] = d.Name
	}
	if addr, err := netip.ParseAddrPort(b.Listen); err != nil || !addr.Addr().Is4() {
		out.add(where, msgBACnetListen, b.Listen)
	} else if owner := seen.listens[b.Listen]; owner != "" && addr.Port() != 0 {
		out.add(where, msgBACnetListenUsed, b.Listen, owner)
	} else {
		seen.listens[b.Listen] = d.Name
	}
	if b.VendorID < 0 || b.VendorID > MaxVendorID {
		out.add(where, msgVendorID, b.VendorID, MaxVendorID)
	}
	if !bacnet.ValidMaxAPDU(b.MaxAPDU) {
		out.add(where, msgMaxAPDU, b.MaxAPDU)
	}
}

func validateBACnetPoint(p *Point, where string, objects map[bacnet.ObjectID]string, out *faults) {
	b := p.BACnet
	objectType, kind, ok := pointType(b.ObjectType)
	if !ok {
		out.add(where, msgObjectType, b.ObjectType, pointTypeNames())
		return
	}
	switch {
	case b.Instance == nil:
		out.add(where, msgNoInstance)
	case *b.Instance < 0 || *b.Instance > MaxInstance:
		out.add(where, msgInstance, *b.Instance, MaxInstance)
	default:
		id := bacnet.ObjectID{Type: objectType, Instance: uint32(*b.Instance)}
		if owner, used := objects[id]; used {
			out.add(where, msgObjectUsed, b.ObjectType, *b.Instance, owner)
		}
		objects[id] = p.Name
	}
	if b.Commandable && bacobj.IsInput(objectType) {
		out.add(where, msgCommandableInput, b.ObjectType)
	}
	analogOnly := b.Min != nil || b.Max != nil || b.Resolution != nil || b.COVIncrement != 0
	switch {
	case kind != bacobj.Analog && b.Units != "":
		out.add(where, msgUnitsKind, b.ObjectType)
	case kind != bacobj.Analog && analogOnly:
		out.add(where, msgAnalogOnly, b.ObjectType)
	case kind == bacobj.Analog:
		if _, known := bacobj.ParseUnits(b.Units); !known {
			out.add(where, msgUnits, b.Units, bacobj.MaxUnits)
		}
		if b.Min != nil && b.Max != nil && *b.Min > *b.Max {
			out.add(where, msgMinAboveMax, *b.Min, *b.Max)
		}
		if b.COVIncrement < 0 {
			out.add(where, msgCOVIncrement, b.COVIncrement)
		}
	}
	if kind != bacobj.Binary && (b.ActiveText != "" || b.InactiveText != "") {
		out.add(where, msgBinaryTexts, b.ObjectType)
	}
	states := len(b.StateTexts)
	constant := p.Behaviour == nil || p.Behaviour.Kind == KindConstant
	switch {
	case kind != bacobj.MultiState && states > 0:
		out.add(where, msgStatesKind, b.ObjectType)
	case kind == bacobj.MultiState && states == 0:
		out.add(where, msgNoStates, b.ObjectType)
	case kind == bacobj.MultiState && constant && (p.Value < 1 || p.Value > float64(states)):
		out.add(where, msgStateValue, p.Value, states, states)
	}
}
