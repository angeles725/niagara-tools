package regmap

import (
	"math"
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

func TestEncodeLaysOutEachTypeInEachByteOrder(t *testing.T) {
	t.Parallel()
	cases := []struct {
		typ   Type
		order Order
		value float64
		want  []uint16
	}{
		// 123456.0 as IEEE 754 single precision is 0x47F12000.
		{Float32, ABCD, 123456, []uint16{0x47F1, 0x2000}},
		{Float32, CDAB, 123456, []uint16{0x2000, 0x47F1}},
		{Float32, BADC, 123456, []uint16{0xF147, 0x0020}},
		{Float32, DCBA, 123456, []uint16{0x0020, 0xF147}},
		{Int32, ABCD, -2, []uint16{0xFFFF, 0xFFFE}},
		{Int32, CDAB, -2, []uint16{0xFFFE, 0xFFFF}},
		{Uint32, ABCD, 305419896, []uint16{0x1234, 0x5678}},
		{Uint32, DCBA, 305419896, []uint16{0x7856, 0x3412}},
		{Int16, ABCD, -2, []uint16{0xFFFE}},
		{Int16, CDAB, -2, []uint16{0xFFFE}},
		{Int16, BADC, -2, []uint16{0xFEFF}},
		{Uint16, DCBA, 0x1234, []uint16{0x3412}},
		{Bit, ABCD, 1, []uint16{1}},
	}
	for _, c := range cases {
		got := Encode(c.typ, c.order, c.value)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Encode(%s, %s, %v) = %04X, want %04X", c.typ, c.order, c.value, got, c.want)
		}
		if back := Decode(c.typ, c.order, got); back != c.value {
			t.Errorf("Decode(%s, %s, %04X) = %v, want %v", c.typ, c.order, got, back, c.value)
		}
	}
}

func TestEncodeRoundsAndSaturatesIntegers(t *testing.T) {
	t.Parallel()
	cases := []struct {
		typ   Type
		value float64
		want  []uint16
	}{
		{Int16, 21.6, []uint16{22}},
		{Int16, -21.6, []uint16{0xFFEA}},
		{Int16, 40000, []uint16{0x7FFF}},
		{Int16, -40000, []uint16{0x8000}},
		{Uint16, -5, []uint16{0}},
		{Uint16, 70000, []uint16{0xFFFF}},
		{Uint32, 5e9, []uint16{0xFFFF, 0xFFFF}},
		{Int32, -5e9, []uint16{0x8000, 0x0000}},
		{Int16, math.NaN(), []uint16{0}},
		{Bit, 0.4, []uint16{1}},
		{Bit, 0, []uint16{0}},
	}
	for _, c := range cases {
		if got := Encode(c.typ, ABCD, c.value); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Encode(%s, %v) = %04X, want %04X", c.typ, c.value, got, c.want)
		}
	}
}

func unit() *Unit {
	dev := model.NewDevice(model.DeviceSpec{Name: "d", Points: []model.PointSpec{
		{Name: "run", Initial: 1, Writable: true},
		{Name: "alarm", Initial: 1},
		{Name: "speed", Initial: 1200, Writable: true},
		{Name: "energy", Initial: 123456, Writable: true},
		{Name: "serial", Initial: 77},
		{Name: "temp", Initial: -2},
	}})
	return NewUnit(dev, []Binding{
		{Point: "run", Table: Coils, Address: 3, Type: Bit, Order: ABCD},
		{Point: "alarm", Table: DiscreteInputs, Address: 0, Type: Bit, Order: ABCD},
		{Point: "speed", Table: HoldingRegisters, Address: 10, Type: Uint16, Order: ABCD},
		{Point: "energy", Table: HoldingRegisters, Address: 11, Type: Float32, Order: CDAB},
		{Point: "serial", Table: HoldingRegisters, Address: 20, Type: Uint16, Order: ABCD},
		{Point: "temp", Table: InputRegisters, Address: 0, Type: Int16, Order: ABCD},
	})
}

func TestResolveNeedsEveryAddressMappedAndWritableForAWrite(t *testing.T) {
	t.Parallel()
	u := unit()
	cases := []struct {
		name  string
		table Table
		addr  uint16
		qty   int
		write bool
		want  []string
		ok    bool
	}{
		{"two adjacent points", HoldingRegisters, 10, 3, false, []string{"speed", "energy"}, true},
		{"one word of a two-word point", HoldingRegisters, 12, 1, false, []string{"energy"}, true},
		{"a range with an unmapped address", HoldingRegisters, 10, 4, false, nil, false},
		{"an unmapped address", HoldingRegisters, 9, 1, false, nil, false},
		{"another table", InputRegisters, 10, 1, false, nil, false},
		{"a write to writable points", HoldingRegisters, 10, 3, true, []string{"speed", "energy"}, true},
		{"a write to a read-only point", HoldingRegisters, 20, 1, true, nil, false},
		{"a read of the read-only point", HoldingRegisters, 20, 1, false, []string{"serial"}, true},
		{"the last address", Coils, 65535, 1, false, nil, false},
	}
	for _, c := range cases {
		got, ok := u.Resolve(c.table, c.addr, c.qty, c.write)
		if ok != c.ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, %v; want %v, %v", c.name, got, ok, c.want, c.ok)
		}
	}
}

func TestReadsEncodeThePointValues(t *testing.T) {
	t.Parallel()
	u := unit()
	if got := u.ReadWords(HoldingRegisters, 10, 3, 0); !reflect.DeepEqual(got, []uint16{1200, 0x2000, 0x47F1}) {
		t.Errorf("holding 10-12 = %04X", got)
	}
	if got := u.ReadWords(InputRegisters, 0, 1, 0); !reflect.DeepEqual(got, []uint16{0xFFFE}) {
		t.Errorf("input 0 = %04X", got)
	}
	if got := u.ReadBits(Coils, 3, 1, 0); !reflect.DeepEqual(got, []bool{true}) {
		t.Errorf("coil 3 = %v", got)
	}
	if got := u.ReadBits(DiscreteInputs, 0, 1, 0); !reflect.DeepEqual(got, []bool{true}) {
		t.Errorf("discrete 0 = %v", got)
	}
}

func TestWritesChangeThePointValues(t *testing.T) {
	t.Parallel()
	u := unit()
	u.WriteWords(10, []uint16{900}, 0)
	if v, _ := u.Device().Get("speed", 0); v != 900 {
		t.Errorf("speed = %v, want 900", v)
	}
	u.WriteBits(3, []bool{false}, 0)
	if v, _ := u.Device().Get("run", 0); v != 0 {
		t.Errorf("run = %v, want 0", v)
	}
	// 1.5 is 0x3FC00000; in CDAB order the low word comes first.
	u.WriteWords(11, []uint16{0x0000, 0x3FC0}, 0)
	if v, _ := u.Device().Get("energy", 0); v != 1.5 {
		t.Errorf("energy = %v, want 1.5", v)
	}
	// Only the high word (address 12 in CDAB order): 0x40400000 is 3.0.
	u.WriteWords(12, []uint16{0x4040}, 0)
	if v, _ := u.Device().Get("energy", 0); v != 3 {
		t.Errorf("energy after one word = %v, want 3", v)
	}
}

func TestReferenceOfIsTheOneBasedNotation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		table   Table
		address int
		want    string
	}{
		{Coils, 0, "00001"}, {DiscreteInputs, 2, "10003"}, {InputRegisters, 0, "30001"},
		{HoldingRegisters, 9998, "49999"}, {HoldingRegisters, 9999, "410000"}, {HoldingRegisters, 65535, "465536"},
	}
	for _, c := range cases {
		if got := c.table.ReferenceOf(c.address); got != c.want {
			t.Errorf("%s address %d = %s, want %s", c.table, c.address, got, c.want)
		}
	}
}
