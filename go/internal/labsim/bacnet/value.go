package bacnet

import (
	"encoding/binary"
	"math"
)

// Kind is the application data type of a Value; it equals the tag number.
type Kind byte

// The application data types.
const (
	KindNull     Kind = tagNull
	KindBool     Kind = tagBoolean
	KindUnsigned Kind = tagUnsigned
	KindSigned   Kind = tagSigned
	KindReal     Kind = tagReal
	KindDouble   Kind = tagDouble
	KindOctets   Kind = tagOctets
	KindString   Kind = tagString
	KindBits     Kind = tagBits
	KindEnum     Kind = tagEnum
	KindDate     Kind = tagDate
	KindTime     Kind = tagTime
	KindOID      Kind = tagOID
)

// Value is one application-tagged value. Only the field of its Kind is set.
type Value struct {
	Kind   Kind
	Bool   bool
	Uint   uint32 // unsigned and enumerated
	Int    int32
	Real   float32
	Double float64
	Text   string   // a character string, UTF-8 (character set 0)
	Bits   []bool   // a bit string, first bit first
	Raw    []byte   // an octet string, or the 4 octets of a date or a time
	Object ObjectID // an object identifier
}

// The constructors of the values the simulator produces.
func Null() Value                 { return Value{Kind: KindNull} }
func Boolean(on bool) Value       { return Value{Kind: KindBool, Bool: on} }
func Unsigned(n uint32) Value     { return Value{Kind: KindUnsigned, Uint: n} }
func Signed(n int32) Value        { return Value{Kind: KindSigned, Int: n} }
func Real(f float32) Value        { return Value{Kind: KindReal, Real: f} }
func Enum(n uint32) Value         { return Value{Kind: KindEnum, Uint: n} }
func Text(s string) Value         { return Value{Kind: KindString, Text: s} }
func OID(id ObjectID) Value       { return Value{Kind: KindOID, Object: id} }
func Bits(bits ...bool) Value     { return Value{Kind: KindBits, Bits: bits} }
func (v Value) IsNull() bool      { return v.Kind == KindNull }
func unsignedOctets(n uint32) int { return max(1, (bitsLen(n)+7)/8) }

func bitsLen(n uint32) int {
	length := 0
	for ; n != 0; n >>= 8 {
		length += 8
	}
	return length
}

// appendUnsigned writes n in as few octets as hold it, most significant first.
func appendUnsigned(b []byte, n uint32) []byte {
	for shift := 8 * (unsignedOctets(n) - 1); shift >= 0; shift -= 8 {
		b = append(b, byte(n>>shift))
	}
	return b
}

// signedOctets is how many octets hold n in two's complement.
func signedOctets(n int32) int {
	for size := 1; size < 4; size++ {
		if limit := int32(1) << (8*size - 1); n >= -limit && n < limit {
			return size
		}
	}
	return 4
}

// packBits packs a bit string: the count of unused bits of the last octet,
// then the bits, first bit in the most significant position.
func packBits(bits []bool) []byte {
	out := make([]byte, 1+(len(bits)+7)/8)
	out[0] = byte((8 - len(bits)%8) % 8)
	for i, on := range bits {
		if on {
			out[1+i/8] |= 0x80 >> (i % 8)
		}
	}
	return out
}

// Append writes the value with its application tag.
func (v Value) Append(b []byte) []byte {
	switch v.Kind {
	case KindBool:
		length := 0
		if v.Bool {
			length = 1
		}
		return appendTag(b, tagBoolean, false, length)
	case KindUnsigned, KindEnum:
		return appendUnsigned(appendTag(b, byte(v.Kind), false, unsignedOctets(v.Uint)), v.Uint)
	case KindSigned:
		size := signedOctets(v.Int)
		b = appendTag(b, tagSigned, false, size)
		for shift := 8 * (size - 1); shift >= 0; shift -= 8 {
			b = append(b, byte(v.Int>>shift))
		}
		return b
	case KindReal:
		return binary.BigEndian.AppendUint32(appendTag(b, tagReal, false, 4), math.Float32bits(v.Real))
	case KindDouble:
		return binary.BigEndian.AppendUint64(appendTag(b, tagDouble, false, 8), math.Float64bits(v.Double))
	case KindString:
		b = append(appendTag(b, tagString, false, len(v.Text)+1), charsetUTF8)
		return append(b, v.Text...)
	case KindBits:
		packed := packBits(v.Bits)
		return append(appendTag(b, tagBits, false, len(packed)), packed...)
	case KindOctets, KindDate, KindTime:
		return append(appendTag(b, byte(v.Kind), false, len(v.Raw)), v.Raw...)
	case KindOID:
		return binary.BigEndian.AppendUint32(appendTag(b, tagOID, false, 4), v.Object.Uint32())
	}
	return appendTag(b, tagNull, false, 0)
}

// AppendCtxUnsigned writes an unsigned or enumerated value with a context tag.
func AppendCtxUnsigned(b []byte, number byte, n uint32) []byte {
	return appendUnsigned(appendTag(b, number, true, unsignedOctets(n)), n)
}

// AppendCtxOID writes an object identifier with a context tag.
func AppendCtxOID(b []byte, number byte, id ObjectID) []byte {
	return binary.BigEndian.AppendUint32(appendTag(b, number, true, 4), id.Uint32())
}

// AppendCtxBool writes a boolean with a context tag: one octet of content.
func AppendCtxBool(b []byte, number byte, on bool) []byte {
	b = appendTag(b, number, true, 1)
	if on {
		return append(b, 1)
	}
	return append(b, 0)
}

// AppendCtxText writes a character string with a context tag.
func AppendCtxText(b []byte, number byte, s string) []byte {
	b = append(appendTag(b, number, true, len(s)+1), charsetUTF8)
	return append(b, s...)
}

// AppendEnclosed writes values between the opening and closing tags number.
func AppendEnclosed(b []byte, number byte, values []Value) []byte {
	b = AppendOpening(b, number)
	for _, v := range values {
		b = v.Append(b)
	}
	return AppendClosing(b, number)
}
