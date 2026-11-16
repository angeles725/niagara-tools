// Package bacobj is the BACnet object model of lab-sim: a Device object and
// analog, binary and multi-state inputs, outputs and values laid over the
// points of a protocol-neutral model device. It answers property reads and
// writes; the wire and the services are in bacnet and bacserver.
//
// A point's field value (what the scenario, a behaviour or the control port's
// set gives it) is the present-value of an object that is not commandable. A
// commandable object keeps a priority array: its present-value is the command
// of the highest priority, shown through the model as an override, and the
// field value underneath is its relinquish-default.
package bacobj

import (
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// Priorities is the length of a priority array.
const Priorities = 16

// DeviceSpec describes the Device object.
type DeviceSpec struct {
	Instance    uint32
	Name        string
	Description string
	VendorID    uint32
	VendorName  string
	Model       string
	Firmware    string
	Application string
	MaxAPDU     int
	COV         bool // SubscribeCOV is served and analog objects have cov-increment
}

// ObjectSpec describes one object and the model point behind it. The object's
// name is the point's.
type ObjectSpec struct {
	ID           bacnet.ObjectID
	Point        string
	Description  string
	Units        uint32
	Commandable  bool // it has a priority array
	Writable     bool // present-value takes a plain write (when not commandable)
	ActiveText   string
	InactiveText string
	StateTexts   []string // multi-state: one per state
	Min, Max     *float64 // analog: min-pres-value and max-pres-value, and the range of a write
	Resolution   *float64
	COVIncrement float64
}

// object is an ObjectSpec with its running state, which the model device's
// lock guards.
type object struct {
	ObjectSpec
	kind     Kind
	table    []property
	priority [Priorities]*float64
}

// Device is one BACnet device. Every method is safe for concurrent use.
type Device struct {
	spec      DeviceSpec
	points    *model.Device
	id        bacnet.ObjectID
	order     []bacnet.ObjectID
	objects   map[bacnet.ObjectID]*object
	supported []int
}

// NewDevice lays the objects over the points of a model device. The specs come
// from a validated scenario: types are known and identifiers unique.
func NewDevice(spec DeviceSpec, objects []ObjectSpec, points *model.Device) *Device {
	d := &Device{spec: spec, points: points, id: bacnet.ObjectID{Type: bacnet.Device, Instance: spec.Instance},
		objects: map[bacnet.ObjectID]*object{}}
	d.order = append(d.order, d.id)
	for _, o := range objects {
		kind, _ := KindOf(o.ID.Type)
		obj := &object{ObjectSpec: o, kind: kind}
		obj.table = d.tableOf(obj)
		d.order = append(d.order, o.ID)
		d.objects[o.ID] = obj
	}
	return d
}

// ID is the identifier of the Device object.
func (d *Device) ID() bacnet.ObjectID { return d.id }

// Spec is the description the device was built from.
func (d *Device) Spec() DeviceSpec { return d.spec }

// Model is the model device behind the objects.
func (d *Device) Model() *model.Device { return d.points }

// SetSupported states which services the server in front of the device
// answers, as bits of protocol-services-supported.
func (d *Device) SetSupported(bits []int) { d.supported = bits }

// Objects lists the object identifiers: the device, then the scenario order.
func (d *Device) Objects() []bacnet.ObjectID { return append([]bacnet.ObjectID(nil), d.order...) }

// resolve finds an object. The device answers to its own identifier and to the
// wildcard instance; obj is nil for it.
func (d *Device) resolve(id bacnet.ObjectID) (obj *object, isDevice, ok bool) {
	if id == d.id || (id.Type == bacnet.Device && id.Instance == bacnet.WildcardInstance) {
		return nil, true, true
	}
	obj = d.objects[id]
	return obj, false, obj != nil
}

// Has reports whether the device has the object.
func (d *Device) Has(id bacnet.ObjectID) bool {
	_, _, ok := d.resolve(id)
	return ok
}

// Point is the model point behind an object; "" for the device and for an
// object it does not have.
func (d *Device) Point(id bacnet.ObjectID) string {
	if obj := d.objects[id]; obj != nil {
		return obj.Point
	}
	return ""
}

// FindByName finds an object by its object-name.
func (d *Device) FindByName(name string) (bacnet.ObjectID, bool) {
	if name == d.spec.Name {
		return d.id, true
	}
	for _, id := range d.order[1:] {
		if d.objects[id].Point == name {
			return id, true
		}
	}
	return bacnet.ObjectID{}, false
}

// COVIncrement is the change of present-value that is worth a notification,
// and whether the object is analog at all (any change of another kind is).
func (d *Device) COVIncrement(id bacnet.ObjectID) (increment float64, analog bool) {
	if obj := d.objects[id]; obj != nil && obj.kind == Analog {
		return obj.COVIncrement, true
	}
	return 0, false
}
