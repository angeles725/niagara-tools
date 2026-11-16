package regmap

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// Binding places one point of a device in a table.
type Binding struct {
	Point   string
	Table   Table
	Address uint16 // 0-based; the first of Type.Words() addresses
	Type    Type
	Order   Order
}

// slot is one address of a table: a binding and which of its words lives there.
type slot struct {
	binding *Binding
	word    int
}

// Unit is the register map of one device (one unit id).
type Unit struct {
	device *model.Device
	tables [tableCount]map[uint16]slot
}

// NewUnit builds the map of dev from its bindings, which must not overlap (a
// scenario is validated before it is built).
func NewUnit(dev *model.Device, bindings []Binding) *Unit {
	u := &Unit{device: dev}
	for t := range u.tables {
		u.tables[t] = map[uint16]slot{}
	}
	for i := range bindings {
		b := &bindings[i]
		for w := range b.Type.Words() {
			u.tables[b.Table][b.Address+uint16(w)] = slot{binding: b, word: w}
		}
	}
	return u
}

// Device is the device behind the map.
func (u *Unit) Device() *model.Device { return u.device }

// Resolve names the points that quantity addresses from address touch, in
// address order. It fails when one of the addresses holds no point (part of a
// two-register point is enough to touch it) or, for a write, when one of the
// points is read-only.
func (u *Unit) Resolve(t Table, address uint16, quantity int, write bool) ([]string, bool) {
	var points []string
	for a := int(address); a < int(address)+quantity; a++ {
		s, ok := u.tables[t][uint16(a)]
		if a > 0xFFFF || !ok {
			return nil, false
		}
		name := s.binding.Point
		if write && !u.device.Writable(name) {
			return nil, false
		}
		if len(points) == 0 || points[len(points)-1] != name {
			points = append(points, name)
		}
	}
	return points, true
}

// ReadWords reads registers of a range Resolve accepted.
func (u *Unit) ReadWords(t Table, address uint16, quantity int, elapsed time.Duration) []uint16 {
	out := make([]uint16, quantity)
	u.device.Do(elapsed, func(tx model.Tx) {
		for i := range out {
			s := u.tables[t][address+uint16(i)]
			b := s.binding
			out[i] = Encode(b.Type, b.Order, tx.Get(b.Point))[s.word]
		}
	})
	return out
}

// ReadBits reads coils or discrete inputs of a range Resolve accepted.
func (u *Unit) ReadBits(t Table, address uint16, quantity int, elapsed time.Duration) []bool {
	out := make([]bool, quantity)
	u.device.Do(elapsed, func(tx model.Tx) {
		for i := range out {
			out[i] = tx.Get(u.tables[t][address+uint16(i)].binding.Point) != 0
		}
	})
	return out
}

// WriteWords writes holding registers of a range Resolve accepted. A write of
// one register of a two-register point changes that half of its value.
func (u *Unit) WriteWords(address uint16, words []uint16, elapsed time.Duration) {
	u.device.Do(elapsed, func(tx model.Tx) {
		for i, w := range words {
			s := u.tables[HoldingRegisters][address+uint16(i)]
			b := s.binding
			current := Encode(b.Type, b.Order, tx.Get(b.Point))
			current[s.word] = w
			tx.Set(b.Point, Decode(b.Type, b.Order, current))
		}
	})
}

// WriteBits writes coils of a range Resolve accepted.
func (u *Unit) WriteBits(address uint16, bits []bool, elapsed time.Duration) {
	u.device.Do(elapsed, func(tx model.Tx) {
		for i, on := range bits {
			value := 0.0
			if on {
				value = 1
			}
			tx.Set(u.tables[Coils][address+uint16(i)].binding.Point, value)
		}
	})
}
