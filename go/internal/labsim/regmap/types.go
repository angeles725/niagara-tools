// Package regmap maps a device's points onto the four Modbus tables: which
// addresses a point occupies, how its value is laid out in registers, and the
// reads and writes of a range of addresses.
package regmap

import (
	"fmt"
	"math"
)

// Table is one of the four Modbus data tables.
type Table int

// The tables, in the order of their reference digit (0, 1, 3, 4).
const (
	Coils Table = iota
	DiscreteInputs
	InputRegisters
	HoldingRegisters
	tableCount
)

// tableInfo is what a scenario and a register map say about each table.
var tableInfo = [tableCount]struct {
	name      string
	reference string // leading digit of the 0xxxx/1xxxx/3xxxx/4xxxx notation
	bits      bool
	writable  bool
}{
	Coils:            {"coil", "0", true, true},
	DiscreteInputs:   {"discrete_input", "1", true, false},
	InputRegisters:   {"input_register", "3", false, false},
	HoldingRegisters: {"holding_register", "4", false, true},
}

// String is the table's name in a scenario file.
func (t Table) String() string { return tableInfo[t].name }

// Reference is the leading digit of the table's 1-based reference notation.
func (t Table) Reference() string { return tableInfo[t].reference }

// Bits reports whether the table holds single bits (coils, discrete inputs).
func (t Table) Bits() bool { return tableInfo[t].bits }

// Writable reports whether the protocol has a write for the table.
func (t Table) Writable() bool { return tableInfo[t].writable }

// ParseTable finds a table by its scenario name.
func ParseTable(name string) (Table, bool) {
	for t := range tableCount {
		if tableInfo[t].name == name {
			return t, true
		}
	}
	return 0, false
}

// Type is how a point's value is encoded.
type Type string

// The encodings.
const (
	Bit     Type = "bit"
	Int16   Type = "int16"
	Uint16  Type = "uint16"
	Int32   Type = "int32"
	Uint32  Type = "uint32"
	Float32 Type = "float32"
)

// typeInfo gives each type its size and the values it can hold.
var typeInfo = map[Type]struct {
	words    int
	min, max float64
}{
	Bit:     {1, 0, 1},
	Int16:   {1, math.MinInt16, math.MaxInt16},
	Uint16:  {1, 0, math.MaxUint16},
	Int32:   {2, math.MinInt32, math.MaxInt32},
	Uint32:  {2, 0, math.MaxUint32},
	Float32: {2, -math.MaxFloat32, math.MaxFloat32},
}

// Known reports whether t is one of the encodings.
func (t Type) Known() bool { _, ok := typeInfo[t]; return ok }

// Words is the number of addresses a point of this type occupies.
func (t Type) Words() int { return typeInfo[t].words }

// Fits reports whether the type can hold v (a bit holds exactly 0 or 1).
func (t Type) Fits(v float64) bool {
	info := typeInfo[t]
	if math.IsNaN(v) || v < info.min || v > info.max {
		return false
	}
	return t != Bit || v == 0 || v == 1
}

// Order is the byte order of a value on the wire, named by where the bytes
// A B C D of the big-endian value go. A 16-bit value keeps its bytes for ABCD
// and CDAB and swaps them for BADC and DCBA.
type Order string

// The four byte and word orders integrators meet.
const (
	ABCD Order = "ABCD" // big-endian: high word first, high byte first
	CDAB Order = "CDAB" // word swap: low word first
	BADC Order = "BADC" // byte swap inside each word
	DCBA Order = "DCBA" // little-endian: both swapped
)

// permutations says which big-endian byte goes in each wire position.
var permutations = map[Order][4]int{
	ABCD: {0, 1, 2, 3},
	CDAB: {2, 3, 0, 1},
	BADC: {1, 0, 3, 2},
	DCBA: {3, 2, 1, 0},
}

// Known reports whether o is one of the four orders.
func (o Order) Known() bool { _, ok := permutations[o]; return ok }

// swapsBytes reports whether the order swaps the two bytes of a register.
func (o Order) swapsBytes() bool { return o == BADC || o == DCBA }

// ReferenceOf is a 0-based address in the 1-based reference notation drivers
// and device manuals use: the table's digit and four digits (40001), or five
// once the address passes 9998 (410000).
func (t Table) ReferenceOf(address int) string {
	if n := address + 1; n > 9999 {
		return fmt.Sprintf("%s%05d", t.Reference(), n)
	}
	return fmt.Sprintf("%s%04d", t.Reference(), address+1)
}
