package bacnet

import (
	"encoding/binary"
	"math"
)

// charsetUTF8 is character set 0: ISO 10646 UTF-8, which older revisions of the
// standard called ANSI X3.4.
const charsetUTF8 = 0

// content decodes the octets of a primitive value of the given kind.
func content(kind Kind, length int, data []byte) (Value, error) {
	v := Value{Kind: kind}
	switch kind {
	case KindNull:
		if length != 0 {
			return v, RejectInvalidTag
		}
	case KindBool:
		if length > 1 {
			return v, RejectInvalidTag
		}
		v.Bool = length == 1
	case KindUnsigned, KindEnum:
		n, err := unsigned(data)
		v.Uint = n
		return v, err
	case KindSigned:
		if len(data) < 1 || len(data) > 4 {
			return v, RejectInvalidTag
		}
		n := int32(int8(data[0]))
		for _, octet := range data[1:] {
			n = n<<8 | int32(octet)
		}
		v.Int = n
	case KindReal:
		if len(data) != 4 {
			return v, RejectInvalidTag
		}
		v.Real = math.Float32frombits(binary.BigEndian.Uint32(data))
	case KindDouble:
		if len(data) != 8 {
			return v, RejectInvalidTag
		}
		v.Double = math.Float64frombits(binary.BigEndian.Uint64(data))
	case KindString:
		if len(data) < 1 {
			return v, RejectInvalidTag
		}
		if data[0] != charsetUTF8 {
			return v, RejectInvalidParameterType
		}
		v.Text = string(data[1:])
	case KindBits:
		if len(data) < 1 || data[0] > 7 || (len(data) == 1 && data[0] != 0) {
			return v, RejectInvalidTag
		}
		if count := 8*(len(data)-1) - int(data[0]); count > 0 {
			v.Bits = make([]bool, count)
		}
		for i := range v.Bits {
			v.Bits[i] = data[1+i/8]&(0x80>>(i%8)) != 0
		}
	case KindDate, KindTime:
		if len(data) != 4 {
			return v, RejectInvalidTag
		}
		v.Raw = append([]byte(nil), data...)
	case KindOctets:
		v.Raw = append([]byte(nil), data...)
	case KindOID:
		if len(data) != 4 {
			return v, RejectInvalidTag
		}
		v.Object = objectID(binary.BigEndian.Uint32(data))
	default:
		return v, RejectInvalidTag
	}
	return v, nil
}

func unsigned(data []byte) (uint32, error) {
	if len(data) < 1 || len(data) > 4 {
		return 0, RejectInvalidTag
	}
	var n uint32
	for _, octet := range data {
		n = n<<8 | uint32(octet)
	}
	return n, nil
}

// Value reads one application-tagged value.
func (r *Reader) Value() (Value, error) {
	t, data, err := r.next()
	if err != nil {
		return Value{}, err
	}
	if t.context {
		return Value{}, RejectInvalidTag
	}
	return content(Kind(t.number), t.length, data)
}

// ctx reads the primitive context tag number as a value of kind.
func (r *Reader) ctx(number byte, kind Kind) (Value, error) {
	t, data, err := r.next()
	if err != nil {
		return Value{}, err
	}
	if !t.context || t.opening || t.closing || t.number != number {
		return Value{}, RejectInvalidTag
	}
	if kind == KindBool { // a context boolean has one octet of content
		if len(data) != 1 {
			return Value{}, RejectInvalidTag
		}
		return Boolean(data[0] != 0), nil
	}
	return content(kind, t.length, data)
}

// CtxUnsigned reads a required unsigned or enumerated context value.
func (r *Reader) CtxUnsigned(number byte) (uint32, error) {
	v, err := r.ctx(number, KindUnsigned)
	return v.Uint, err
}

// OptCtxUnsigned reads an optional unsigned or enumerated context value.
func (r *Reader) OptCtxUnsigned(number byte) (uint32, bool, error) {
	if r.Done() {
		return 0, false, nil
	}
	if _, _, err := r.head(); err != nil {
		return 0, false, err // a tag that is cut short is not an absent one
	}
	if !r.AtContext(number) {
		return 0, false, nil
	}
	n, err := r.CtxUnsigned(number)
	return n, err == nil, err
}

// CtxOID reads a required object identifier with a context tag.
func (r *Reader) CtxOID(number byte) (ObjectID, error) {
	v, err := r.ctx(number, KindOID)
	return v.Object, err
}

// CtxBool reads a required boolean with a context tag.
func (r *Reader) CtxBool(number byte) (bool, error) {
	v, err := r.ctx(number, KindBool)
	return v.Bool, err
}

// CtxText reads a required character string with a context tag.
func (r *Reader) CtxText(number byte) (string, error) {
	v, err := r.ctx(number, KindString)
	return v.Text, err
}

// Enclosed reads the application values between the opening and closing tags
// number.
func (r *Reader) Enclosed(number byte) ([]Value, error) {
	if err := r.Open(number); err != nil {
		return nil, err
	}
	var values []Value
	for !r.AtClosing(number) {
		if t, _, err := r.head(); err == nil && t.closing {
			return nil, RejectInvalidTag // closed by another tag
		}
		v, err := r.Value()
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, r.Close(number)
}
