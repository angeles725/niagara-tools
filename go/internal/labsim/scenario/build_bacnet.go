package scenario

import (
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// BACnetEnd is one BACnet device of a scenario and where it listens.
type BACnetEnd struct {
	Name   string // the scenario name of the device
	Listen string
	Device *bacobj.Device
}

// ObjectID is the object identifier of a mapped point of a validated scenario.
func (p *Point) ObjectID() bacnet.ObjectID {
	t, _ := bacnet.ParseObjectType(p.BACnet.ObjectType)
	return bacnet.ObjectID{Type: t, Instance: uint32(*p.BACnet.Instance)}
}

// bacnetEnd lays the BACnet objects of a device over its model points.
func bacnetEnd(d *Device, points *model.Device) BACnetEnd {
	b := d.BACnet
	spec := bacobj.DeviceSpec{Instance: uint32(*b.DeviceInstance), Name: d.Name, Description: b.Description,
		VendorID: uint32(b.VendorID), VendorName: b.VendorName, Model: b.Model, Firmware: b.Firmware,
		Application: b.Application, MaxAPDU: b.MaxAPDU, COV: b.COV}
	var objects []bacobj.ObjectSpec
	for i := range d.Points {
		p := &d.Points[i]
		m := p.BACnet
		if m == nil {
			continue
		}
		id := p.ObjectID()
		units, _ := bacobj.ParseUnits(m.Units)
		objects = append(objects, bacobj.ObjectSpec{ID: id, Point: p.Name, Description: m.Description, Units: units,
			Commandable: m.Commandable, Writable: p.Writable && !bacobj.IsInput(id.Type),
			ActiveText: m.ActiveText, InactiveText: m.InactiveText, StateTexts: m.StateTexts,
			Min: m.Min, Max: m.Max, Resolution: m.Resolution, COVIncrement: m.COVIncrement})
	}
	return BACnetEnd{Name: d.Name, Listen: b.Listen, Device: bacobj.NewDevice(spec, objects, points)}
}
