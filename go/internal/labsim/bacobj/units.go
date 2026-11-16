package bacobj

import (
	"slices"
	"strconv"
)

// MaxUnits is the highest value of the engineering units enumeration.
const MaxUnits = 65535

// NoUnits is the units of an analog object that states none.
const NoUnits = 95

// unitNames are the engineering units (BACnetEngineeringUnits) a scenario can
// name; any other is written as its number.
var unitNames = map[string]uint32{
	"milliamperes": 2, "amperes": 3, "ohms": 4, "volts": 5, "kilovolts": 6, "volt-amperes": 8, "kilovolt-amperes": 9,
	"power-factor": 15, "joules": 16, "kilojoules": 17, "watt-hours": 18, "kilowatt-hours": 19, "hertz": 27,
	"percent-relative-humidity": 29, "millimeters": 30, "meters": 31, "watts": 47, "kilowatts": 48, "megawatts": 49,
	"pascals": 53, "kilopascals": 54, "bars": 55, "degrees-celsius": 62, "degrees-kelvin": 63, "degrees-fahrenheit": 64,
	"hours": 71, "minutes": 72, "seconds": 73, "meters-per-second": 74, "liters-per-second": 87, "liters-per-minute": 88,
	"no-units": NoUnits, "parts-per-million": 96, "percent": 98, "revolutions-per-minute": 104,
}

// ParseUnits reads engineering units: one of the names, or the number of the
// enumeration.
func ParseUnits(text string) (uint32, bool) {
	if n, ok := unitNames[text]; ok {
		return n, true
	}
	n, err := strconv.ParseUint(text, 10, 32)
	return uint32(n), err == nil && n <= MaxUnits
}

// UnitsName is the name of engineering units, or their number.
func UnitsName(units uint32) string {
	for name, n := range unitNames {
		if n == units {
			return name
		}
	}
	return strconv.FormatUint(uint64(units), 10)
}

// UnitNames lists the names ParseUnits knows.
func UnitNames() []string {
	names := make([]string, 0, len(unitNames))
	for name := range unitNames {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
