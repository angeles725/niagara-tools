package bacobj

import (
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// reading is what a property is read from: the device, the object (nil for the
// Device object) and the model at one instant.
type reading struct {
	d  *Device
	o  *object
	tx model.Tx
}

// property is one row of a property table.
type property struct {
	id       bacnet.Property
	required bool // by the standard for the object type; the others are optional
	array    bool
	has      func(d *Device, o *object) bool // nil: every object of the table has it
	get      func(r reading) []bacnet.Value  // the whole value: every element of an array
}

func one(v bacnet.Value) []bacnet.Value { return []bacnet.Value{v} }

func real(f float64) []bacnet.Value { return one(bacnet.Real(float32(f))) }

// noFlags is status-flags with in-alarm, fault, overridden and out-of-service
// all clear.
var noFlags = bacnet.Bits(false, false, false, false)

// common are the properties of every point object, before those of its kind.
var common = []property{
	{id: bacnet.PropObjectIdentifier, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.OID(r.o.ID)) }},
	{id: bacnet.PropObjectName, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Text(r.o.Point)) }},
	{id: bacnet.PropObjectType, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Enum(uint32(r.o.ID.Type))) }},
	{id: bacnet.PropPresentValue, required: true, get: func(r reading) []bacnet.Value { return one(r.o.present(r.tx.Get(r.o.Point))) }},
	{id: bacnet.PropDescription, has: func(_ *Device, o *object) bool { return o.Description != "" },
		get: func(r reading) []bacnet.Value { return one(bacnet.Text(r.o.Description)) }},
	{id: bacnet.PropStatusFlags, required: true, get: func(reading) []bacnet.Value { return one(noFlags) }},
	{id: bacnet.PropEventState, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Enum(bacnet.EventStateNormal)) }},
	{id: bacnet.PropReliability, get: func(reading) []bacnet.Value { return one(bacnet.Enum(bacnet.ReliabilityNoFault)) }},
	{id: bacnet.PropOutOfService, required: true, get: func(reading) []bacnet.Value { return one(bacnet.Boolean(false)) }},
}

// byKind are the properties each kind adds.
var byKind = map[Kind][]property{
	Analog: {
		{id: bacnet.PropUnits, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Enum(r.o.Units)) }},
		{id: bacnet.PropMinPresValue, has: func(_ *Device, o *object) bool { return o.Min != nil },
			get: func(r reading) []bacnet.Value { return real(*r.o.Min) }},
		{id: bacnet.PropMaxPresValue, has: func(_ *Device, o *object) bool { return o.Max != nil },
			get: func(r reading) []bacnet.Value { return real(*r.o.Max) }},
		{id: bacnet.PropResolution, has: func(_ *Device, o *object) bool { return o.Resolution != nil },
			get: func(r reading) []bacnet.Value { return real(*r.o.Resolution) }},
		{id: bacnet.PropCOVIncrement, has: func(d *Device, _ *object) bool { return d.spec.COV },
			get: func(r reading) []bacnet.Value { return real(r.o.COVIncrement) }},
	},
	Binary: {
		{id: bacnet.PropPolarity, required: true, has: func(_ *Device, o *object) bool { return o.ID.Type != bacnet.BinaryValue },
			get: func(reading) []bacnet.Value { return one(bacnet.Enum(bacnet.PolarityNormal)) }},
		{id: bacnet.PropInactiveText, has: func(_ *Device, o *object) bool { return o.InactiveText != "" },
			get: func(r reading) []bacnet.Value { return one(bacnet.Text(r.o.InactiveText)) }},
		{id: bacnet.PropActiveText, has: func(_ *Device, o *object) bool { return o.ActiveText != "" },
			get: func(r reading) []bacnet.Value { return one(bacnet.Text(r.o.ActiveText)) }},
	},
	MultiState: {
		{id: bacnet.PropNumberOfStates, required: true, get: func(r reading) []bacnet.Value { return one(bacnet.Unsigned(uint32(len(r.o.StateTexts)))) }},
		{id: bacnet.PropStateText, array: true, get: func(r reading) []bacnet.Value {
			out := make([]bacnet.Value, 0, len(r.o.StateTexts))
			for _, text := range r.o.StateTexts {
				out = append(out, bacnet.Text(text))
			}
			return out
		}},
	},
}

// commandable are the properties of an object with a priority array.
var commandable = []property{
	{id: bacnet.PropPriorityArray, required: true, array: true, get: func(r reading) []bacnet.Value {
		out := make([]bacnet.Value, 0, Priorities)
		for _, slot := range r.o.priority {
			if slot == nil {
				out = append(out, bacnet.Null())
			} else {
				out = append(out, r.o.present(*slot))
			}
		}
		return out
	}},
	{id: bacnet.PropRelinquishDefault, required: true, get: func(r reading) []bacnet.Value { return one(r.o.present(r.tx.Field(r.o.Point))) }},
}

// tableOf lists the properties an object has, in the order ALL gives them.
func (d *Device) tableOf(o *object) []property {
	candidates := append(append([]property(nil), common...), byKind[o.kind]...)
	if o.Commandable {
		candidates = append(candidates, commandable...)
	}
	var table []property
	for _, p := range candidates {
		if p.has == nil || p.has(d, o) {
			table = append(table, p)
		}
	}
	return table
}
