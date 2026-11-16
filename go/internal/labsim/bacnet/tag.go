// Package bacnet is the BACnet wire codec of lab-sim: the BACnet/IP virtual
// link layer (BVLL), the network layer header (NPDU), the application layer
// PDUs, the tag encoding of clause 20.2 of ANSI/ASHRAE 135 and the service
// requests and acknowledgements the simulator answers. It was written from the
// rules of the public standard and has not been validated against another
// BACnet implementation.
package bacnet

// tag is one decoded tag header.
type tag struct {
	number  byte
	context bool
	opening bool
	closing bool
	length  int // octets of content; for an application boolean, its value
}

// Tag numbers of the application data types (clause 20.2.1.4).
const (
	tagNull = iota
	tagBoolean
	tagUnsigned
	tagSigned
	tagReal
	tagDouble
	tagOctets
	tagString
	tagBits
	tagEnum
	tagDate
	tagTime
	tagOID
)

const (
	classContext = 0x08
	lvtExtended  = 5
	lvtOpening   = 6
	lvtClosing   = 7
)

// appendHead writes the initial octet of a tag, and the tag number octet for a
// number of 15 or more.
func appendHead(b []byte, number, class, lvt byte) []byte {
	if number < 15 {
		return append(b, number<<4|class|lvt)
	}
	return append(b, 0xF0|class|lvt, number)
}

// appendTag writes the header of a primitive tag whose content is length octets.
func appendTag(b []byte, number byte, context bool, length int) []byte {
	class := byte(0)
	if context {
		class = classContext
	}
	switch {
	case length <= 4:
		return appendHead(b, number, class, byte(length))
	case length <= 253:
		return append(appendHead(b, number, class, lvtExtended), byte(length))
	case length <= 0xFFFF:
		return append(appendHead(b, number, class, lvtExtended), 254, byte(length>>8), byte(length))
	}
	return append(appendHead(b, number, class, lvtExtended), 255, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
}

// AppendOpening writes the opening tag of a constructed value.
func AppendOpening(b []byte, number byte) []byte {
	return appendHead(b, number, classContext, lvtOpening)
}

// AppendClosing writes the closing tag of a constructed value.
func AppendClosing(b []byte, number byte) []byte {
	return appendHead(b, number, classContext, lvtClosing)
}

// Reader decodes tagged data front to back. Every failure is a Reject: the
// reason a device gives for refusing the request.
type Reader struct {
	b []byte
}

// NewReader reads the service data b.
func NewReader(b []byte) *Reader { return &Reader{b: b} }

// Done reports whether everything was read.
func (r *Reader) Done() bool { return len(r.b) == 0 }

// head decodes the next tag header without consuming it and returns its size.
func (r *Reader) head() (tag, int, error) {
	b := r.b
	if len(b) == 0 {
		return tag{}, 0, RejectMissingParameter
	}
	t := tag{number: b[0] >> 4, context: b[0]&classContext != 0}
	lvt, size := int(b[0]&7), 1
	if t.number == 15 {
		if len(b) < 2 || b[1] == 255 {
			return tag{}, 0, RejectInvalidTag
		}
		t.number, size = b[1], 2
	}
	switch {
	case t.context && lvt == lvtOpening:
		t.opening = true
	case t.context && lvt == lvtClosing:
		t.closing = true
	case !t.context && t.number == tagBoolean:
		t.length = lvt
		return t, size, nil // the value is in the tag: no content follows
	case lvt > lvtExtended:
		return tag{}, 0, RejectInvalidTag
	case lvt < lvtExtended:
		t.length = lvt
	default:
		extra := 0
		if len(b) > size {
			t.length, extra = int(b[size]), 1
			if b[size] >= 254 {
				extra = 3 + 2*int(b[size]-254)
			}
		}
		if extra == 0 || len(b) < size+extra {
			return tag{}, 0, RejectInvalidTag
		}
		if extra > 1 {
			t.length = 0
			for _, octet := range b[size+1 : size+extra] {
				t.length = t.length<<8 | int(octet)
			}
		}
		size += extra
	}
	if t.length < 0 || t.length > len(b)-size {
		return tag{}, 0, RejectInvalidTag
	}
	return t, size, nil
}

// next consumes the next tag and returns it with its content.
func (r *Reader) next() (tag, []byte, error) {
	t, size, err := r.head()
	if err != nil {
		return t, nil, err
	}
	content := 0
	if t.context || t.number != tagBoolean {
		content = t.length
	}
	data := r.b[size : size+content]
	r.b = r.b[size+content:]
	return t, data, nil
}

// at reports whether the next tag is the context tag number of the given form.
func (r *Reader) at(number byte, opening, closing bool) bool {
	t, _, err := r.head()
	return err == nil && t.context && t.number == number && t.opening == opening && t.closing == closing
}

// AtContext reports whether a primitive context tag number comes next.
func (r *Reader) AtContext(number byte) bool { return r.at(number, false, false) }

// AtOpening reports whether the opening tag number comes next.
func (r *Reader) AtOpening(number byte) bool { return r.at(number, true, false) }

// AtClosing reports whether the closing tag number comes next.
func (r *Reader) AtClosing(number byte) bool { return r.at(number, false, true) }

// bracket consumes the opening or the closing tag number.
func (r *Reader) bracket(number byte, closing bool) error {
	t, _, err := r.next()
	if err != nil {
		return err
	}
	if !t.context || t.number != number || t.opening == closing || t.closing != closing {
		return RejectInvalidTag
	}
	return nil
}

// Open consumes the opening tag number.
func (r *Reader) Open(number byte) error { return r.bracket(number, false) }

// Close consumes the closing tag number.
func (r *Reader) Close(number byte) error { return r.bracket(number, true) }
