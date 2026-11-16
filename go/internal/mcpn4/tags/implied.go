package tags

import "strings"

// letterTypes maps the type letter ending each key of an impliedTags answer to the value's type
// (entityIo JsonEntityEncoder.key; BajaScript baja/tag/entityDecoder). "N" is a non-finite
// Double or Float, whose value names it ("dNaN", "fpInf").
var letterTypes = map[byte]string{
	'M': "baja:Marker", 'S': "baja:String", 'D': "baja:Double", 'N': "baja:Double", 'I': "baja:Integer",
	'B': "baja:Boolean", 'F': "baja:Float", 'L': "baja:Long", 'A': "baja:AbsTime", 'O': "baja:Ord",
	'Y': "baja:DynamicEnum", 'G': "baja:EnumRange", 'R': "baja:RelTime", 'Z': "baja:TimeZone",
	'U': "baja:Unit", 'J': "baja:String",
}

// Implied is one implied tag: its id, value type and the value as the station encoded it.
type Implied struct {
	ID, Type string
	Value    any
}

// DecodeImplied is one key of an impliedTags answer: the id and type of "n:pointM", ok false
// for a key without a known type letter (a newer station: reported, never guessed).
func DecodeImplied(key string, value any) (Implied, bool) {
	if len(key) < 2 {
		return Implied{}, false
	}
	t, ok := letterTypes[key[len(key)-1]]
	if !ok {
		return Implied{}, false
	}
	if key[len(key)-1] == 'N' && strings.HasPrefix(stringOf(value), "f") {
		t = "baja:Float"
	}
	return Implied{ID: key[:len(key)-1], Type: t, Value: value}, true
}

func stringOf(v any) string {
	s, _ := v.(string)
	return s
}
