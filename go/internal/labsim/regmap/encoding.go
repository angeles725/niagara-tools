package regmap

import (
	"encoding/binary"
	"math"
)

// Encode lays a value out in registers. An integer type rounds to the nearest
// integer and saturates at its limits (NaN is 0); a bit is 1 for any non-zero
// value.
func Encode(t Type, o Order, v float64) []uint16 {
	switch t {
	case Bit:
		if v != 0 {
			return []uint16{1}
		}
		return []uint16{0}
	case Int16:
		return word(uint16(int16(saturate(t, v))), o)
	case Uint16:
		return word(uint16(saturate(t, v)), o)
	case Int32:
		return pair(uint32(int32(saturate(t, v))), o)
	case Uint32:
		return pair(uint32(saturate(t, v)), o)
	}
	return pair(math.Float32bits(float32(v)), o)
}

// Decode reads the value Encode laid out.
func Decode(t Type, o Order, words []uint16) float64 {
	switch t {
	case Bit:
		if words[0] != 0 {
			return 1
		}
		return 0
	case Int16:
		return float64(int16(word(words[0], o)[0]))
	case Uint16:
		return float64(word(words[0], o)[0])
	}
	raw := unpair(words, o)
	switch t {
	case Int32:
		return float64(int32(raw))
	case Uint32:
		return float64(raw)
	}
	return float64(math.Float32frombits(raw))
}

// saturate rounds v and holds it within the type's limits.
func saturate(t Type, v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	info := typeInfo[t]
	return math.Min(info.max, math.Max(info.min, math.Round(v)))
}

// word is a 16-bit value as one register; it is its own inverse.
func word(v uint16, o Order) []uint16 {
	if o.swapsBytes() {
		v = v<<8 | v>>8
	}
	return []uint16{v}
}

// pair is a 32-bit value as two registers in the given order.
func pair(v uint32, o Order) []uint16 {
	var big, wire [4]byte
	binary.BigEndian.PutUint32(big[:], v)
	for i, from := range permutations[o] {
		wire[i] = big[from]
	}
	return []uint16{binary.BigEndian.Uint16(wire[0:2]), binary.BigEndian.Uint16(wire[2:4])}
}

// unpair reads back what pair wrote.
func unpair(words []uint16, o Order) uint32 {
	var big, wire [4]byte
	binary.BigEndian.PutUint16(wire[0:2], words[0])
	binary.BigEndian.PutUint16(wire[2:4], words[1])
	for i, from := range permutations[o] {
		big[from] = wire[i]
	}
	return binary.BigEndian.Uint32(big[:])
}
