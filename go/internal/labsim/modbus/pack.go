package modbus

import "encoding/binary"

// packBits packs coils eight to a byte, the first one in the least significant
// bit; the unused bits of the last byte are zero.
func packBits(bits []bool) []byte {
	out := make([]byte, (len(bits)+7)/8)
	for i, on := range bits {
		if on {
			out[i/8] |= 1 << (i % 8)
		}
	}
	return out
}

// unpackBits reads the first n coils of packed.
func unpackBits(packed []byte, n int) []bool {
	out := make([]bool, n)
	for i := range out {
		out[i] = packed[i/8]&(1<<(i%8)) != 0
	}
	return out
}

// packWords writes each register high byte first.
func packWords(words []uint16) []byte {
	out := make([]byte, 0, 2*len(words))
	for _, w := range words {
		out = binary.BigEndian.AppendUint16(out, w)
	}
	return out
}

// unpackWords reads registers from an even number of bytes.
func unpackWords(data []byte) []uint16 {
	out := make([]uint16, len(data)/2)
	for i := range out {
		out[i] = binary.BigEndian.Uint16(data[2*i:])
	}
	return out
}
