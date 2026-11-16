package bacobj

import (
	"math"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

// Kind is what the present-value of an object type is.
type Kind int

// The kinds.
const (
	Analog     Kind = iota // REAL
	Binary                 // inactive (0) or active (1)
	MultiState             // a state from 1
)

// typeInfo says, for each object type the simulator has, its kind and whether
// it is an input (never written from the protocol).
var typeInfo = map[bacnet.ObjectType]struct {
	kind  Kind
	input bool
}{
	bacnet.AnalogInput: {Analog, true}, bacnet.AnalogOutput: {Analog, false}, bacnet.AnalogValue: {Analog, false},
	bacnet.BinaryInput: {Binary, true}, bacnet.BinaryOutput: {Binary, false}, bacnet.BinaryValue: {Binary, false},
	bacnet.MultiStateInput: {MultiState, true}, bacnet.MultiStateOutput: {MultiState, false}, bacnet.MultiStateValue: {MultiState, false},
}

// KindOf is the kind of an object type; ok is false for a type that is not a
// point object (the Device, or one the simulator has not).
func KindOf(t bacnet.ObjectType) (Kind, bool) {
	info, ok := typeInfo[t]
	return info.kind, ok
}

// IsInput reports whether t is one of the three input types.
func IsInput(t bacnet.ObjectType) bool { return typeInfo[t].input }

// supportedTypes are the bits of protocol-object-types-supported.
func supportedTypes() []bool {
	out := make([]bool, bacnet.ObjectTypesSupportedBits)
	out[bacnet.Device] = true
	for t := range typeInfo {
		out[t] = true
	}
	return out
}

// present is the present-value of the object for a model value: a REAL, a
// binary state, or a state number held within 1 and the number of states.
func (o *object) present(v float64) bacnet.Value {
	switch o.kind {
	case Binary:
		if v != 0 {
			return bacnet.Enum(1)
		}
		return bacnet.Enum(0)
	case MultiState:
		state := math.Round(v)
		return bacnet.Unsigned(uint32(math.Min(float64(len(o.StateTexts)), math.Max(1, state))))
	}
	return bacnet.Real(float32(v))
}

// accept turns a written value into a model value, or says why it cannot be
// the object's present-value.
func (o *object) accept(v bacnet.Value) (float64, *bacnet.Error) {
	switch o.kind {
	case Binary:
		if v.Kind != bacnet.KindEnum {
			return 0, bacnet.ErrInvalidDataType
		}
		if v.Uint > 1 {
			return 0, bacnet.ErrValueOutOfRange
		}
		return float64(v.Uint), nil
	case MultiState:
		if v.Kind != bacnet.KindUnsigned {
			return 0, bacnet.ErrInvalidDataType
		}
		if v.Uint < 1 || int(v.Uint) > len(o.StateTexts) {
			return 0, bacnet.ErrValueOutOfRange
		}
		return float64(v.Uint), nil
	}
	if v.Kind != bacnet.KindReal {
		return 0, bacnet.ErrInvalidDataType
	}
	f := float64(v.Real)
	if math.IsNaN(f) || math.IsInf(f, 0) || (o.Min != nil && f < *o.Min) || (o.Max != nil && f > *o.Max) {
		return 0, bacnet.ErrValueOutOfRange
	}
	return f, nil
}
