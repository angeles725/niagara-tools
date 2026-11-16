package bacnet

import "fmt"

// ObjectType is a BACnet object type (BACnetObjectType).
type ObjectType uint16

// The object types the simulator has.
const (
	AnalogInput      ObjectType = 0
	AnalogOutput     ObjectType = 1
	AnalogValue      ObjectType = 2
	BinaryInput      ObjectType = 3
	BinaryOutput     ObjectType = 4
	BinaryValue      ObjectType = 5
	Device           ObjectType = 8
	MultiStateInput  ObjectType = 13
	MultiStateOutput ObjectType = 14
	MultiStateValue  ObjectType = 19
)

// MaxInstance is the highest instance number an object can have; 4194303 is
// the wildcard.
const (
	MaxInstance      = 4194302
	WildcardInstance = 4194303
)

// objectTypeNames names the object types, as the standard writes them.
var objectTypeNames = map[ObjectType]string{
	AnalogInput: "analog-input", AnalogOutput: "analog-output", AnalogValue: "analog-value",
	BinaryInput: "binary-input", BinaryOutput: "binary-output", BinaryValue: "binary-value",
	Device:          "device",
	MultiStateInput: "multi-state-input", MultiStateOutput: "multi-state-output", MultiStateValue: "multi-state-value",
}

// String is the type's name, or its number for a type the simulator has not.
func (t ObjectType) String() string {
	if name, ok := objectTypeNames[t]; ok {
		return name
	}
	return fmt.Sprintf("object-type-%d", uint16(t))
}

// ParseObjectType finds an object type by name.
func ParseObjectType(name string) (ObjectType, bool) {
	for t, n := range objectTypeNames {
		if n == name {
			return t, true
		}
	}
	return 0, false
}

// ObjectTypeNames lists the names ParseObjectType knows, in type order.
func ObjectTypeNames() []string { return sortedNames(objectTypeNames) }

// ObjectID is an object identifier: a type and an instance number.
type ObjectID struct {
	Type     ObjectType
	Instance uint32
}

// Uint32 packs the identifier: 10 bits of type, 22 of instance.
func (id ObjectID) Uint32() uint32 { return uint32(id.Type)<<22 | id.Instance&0x3FFFFF }

func objectID(packed uint32) ObjectID {
	return ObjectID{Type: ObjectType(packed >> 22), Instance: packed & 0x3FFFFF}
}

// String is "type,instance", the notation of the request log and the probe.
func (id ObjectID) String() string { return fmt.Sprintf("%s,%d", id.Type, id.Instance) }
