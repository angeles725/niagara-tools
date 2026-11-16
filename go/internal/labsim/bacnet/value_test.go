package bacnet

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

// unhex reads a frame written as hexadecimal with spaces.
func unhex(t *testing.T, text string) []byte {
	t.Helper()
	b, err := hex.DecodeString(strings.ReplaceAll(text, " ", ""))
	if err != nil {
		t.Fatalf("bad hexadecimal %q: %v", text, err)
	}
	return b
}

// The frames marked "20.2.x" are the encoding examples of clause 20.2 of
// ANSI/ASHRAE 135; the others were built here from the rules of that clause.
var values = []struct {
	name  string
	value Value
	wire  string
}{
	{"null (20.2.2)", Null(), "00"},
	{"boolean false (20.2.3)", Boolean(false), "10"},
	{"boolean true", Boolean(true), "11"},
	{"unsigned 72 (20.2.4)", Unsigned(72), "21 48"},
	{"unsigned 0", Unsigned(0), "21 00"},
	{"unsigned 1024", Unsigned(1024), "22 04 00"},
	{"unsigned 65536", Unsigned(65536), "23 01 00 00"},
	{"unsigned 4294967295", Unsigned(4294967295), "24 FF FF FF FF"},
	{"signed 72 (20.2.5)", Signed(72), "31 48"},
	{"signed -1", Signed(-1), "31 FF"},
	{"signed -129", Signed(-129), "32 FF 7F"},
	{"signed 128", Signed(128), "32 00 80"},
	{"real 72.3 (20.2.6)", Real(72.3), "44 42 90 99 9A"},
	{"real 180", Real(180), "44 43 34 00 00"},
	{"double 72.3 (20.2.7)", Value{Kind: KindDouble, Double: 72.3}, "55 08 40 52 13 33 33 33 33 33"},
	{"octet string (20.2.8)", Value{Kind: KindOctets, Raw: []byte{0x12, 0x34, 0xFF}}, "63 12 34 FF"},
	{"character string (20.2.9)", Text("This is a BACnet string!"),
		"75 19 00 54 68 69 73 20 69 73 20 61 20 42 41 43 6E 65 74 20 73 74 72 69 6E 67 21"},
	{"empty character string", Text(""), "71 00"},
	{"character string beyond ASCII", Text("20 °C"), "75 07 00 32 30 20 C2 B0 43"},
	{"bit string 10101 (20.2.10)", Bits(true, false, true, false, true), "82 03 A8"},
	{"empty bit string", Bits(), "81 00"},
	{"bit string of 8", Bits(true, true, false, false, false, false, false, true), "82 00 C1"},
	{"enumerated 0 (20.2.11)", Enum(0), "91 00"},
	{"enumerated 3", Enum(3), "91 03"},
	{"date 24 January 1991 (20.2.12)", Value{Kind: KindDate, Raw: []byte{0x5B, 0x01, 0x18, 0x04}}, "A4 5B 01 18 04"},
	{"time 17:35:45.17 (20.2.13)", Value{Kind: KindTime, Raw: []byte{0x11, 0x23, 0x2D, 0x11}}, "B4 11 23 2D 11"},
	{"object identifier binary-input 15 (20.2.14)", OID(ObjectID{BinaryInput, 15}), "C4 00 C0 00 0F"},
	{"object identifier device 4194303", OID(ObjectID{Device, 4194303}), "C4 02 3F FF FF"},
}

func TestApplicationValuesEncodeAndDecode(t *testing.T) {
	t.Parallel()
	for _, c := range values {
		wire := unhex(t, c.wire)
		if got := c.value.Append(nil); !bytes.Equal(got, wire) {
			t.Errorf("%s: encoded % X, want % X", c.name, got, wire)
		}
		r := NewReader(wire)
		got, err := r.Value()
		if err != nil || !r.Done() || !reflect.DeepEqual(got, c.value) {
			t.Errorf("%s: decoded %+v, %v (all read: %v); want %+v", c.name, got, err, r.Done(), c.value)
		}
	}
}

func TestALongStringUsesAnExtendedLength(t *testing.T) {
	t.Parallel()
	for _, n := range []int{4, 5, 252, 253, 254, 300, 70000} {
		text := strings.Repeat("a", n)
		wire := Text(text).Append(nil)
		content := n + 1 // the character set octet
		var head []byte
		switch {
		case content <= 4:
			head = []byte{0x70 | byte(content)}
		case content <= 253:
			head = []byte{0x75, byte(content)}
		case content <= 65535:
			head = []byte{0x75, 254, byte(content >> 8), byte(content)}
		default:
			head = []byte{0x75, 255, byte(content >> 24), byte(content >> 16), byte(content >> 8), byte(content)}
		}
		if !bytes.HasPrefix(wire, append(head, 0)) || len(wire) != len(head)+content {
			t.Errorf("%d characters: header % X (%d bytes in all), want % X", n, wire[:len(head)+1], len(wire), head)
		}
		if got, err := NewReader(wire).Value(); err != nil || got.Text != text {
			t.Errorf("%d characters: decoded %d characters, %v", n, len(got.Text), err)
		}
	}
}

func TestContextTagsEncodeAndDecode(t *testing.T) {
	t.Parallel()
	var b []byte
	b = AppendCtxOID(b, 0, ObjectID{AnalogInput, 5})
	b = AppendCtxUnsigned(b, 1, 85)
	b = AppendCtxUnsigned(b, 27, 300)
	b = AppendCtxBool(b, 2, true)
	b = AppendOpening(b, 3)
	b = Real(72.3).Append(b)
	b = AppendClosing(b, 3)
	b = AppendOpening(b, 20)
	b = AppendClosing(b, 20)
	want := unhex(t, "0C 00 00 00 05 19 55 FA 1B 01 2C 29 01 3E 44 42 90 99 9A 3F FE 14 FF 14")
	if !bytes.Equal(b, want) {
		t.Fatalf("encoded % X\nwant    % X", b, want)
	}
	r := NewReader(b)
	if oid, err := r.CtxOID(0); err != nil || oid != (ObjectID{AnalogInput, 5}) {
		t.Errorf("context 0 = %v, %v", oid, err)
	}
	if _, ok, err := r.OptCtxUnsigned(4); ok || err != nil {
		t.Errorf("an absent optional context 4 was found (%v)", err)
	}
	if u, err := r.CtxUnsigned(1); err != nil || u != 85 {
		t.Errorf("context 1 = %d, %v", u, err)
	}
	if u, ok, err := r.OptCtxUnsigned(27); err != nil || !ok || u != 300 {
		t.Errorf("context 27 = %d, %v, %v", u, ok, err)
	}
	if on, err := r.CtxBool(2); err != nil || !on {
		t.Errorf("context 2 = %v, %v", on, err)
	}
	list, err := r.Enclosed(3)
	if err != nil || len(list) != 1 || list[0].Real != 72.3 {
		t.Errorf("enclosed 3 = %+v, %v", list, err)
	}
	if list, err := r.Enclosed(20); err != nil || len(list) != 0 || !r.Done() {
		t.Errorf("empty enclosed 20 = %+v, %v, all read %v", list, err, r.Done())
	}
}

func TestDecodingRefusesWhatIsNotWellFormed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		wire string
		read func(r *Reader) error
		want Reject
	}{
		{"nothing where a value is required", "", func(r *Reader) error { _, err := r.Value(); return err }, RejectMissingParameter},
		{"a value cut short", "44 42 90", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"an extended length cut short", "75 FE 01", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"a real of 3 octets", "43 42 90 99", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"an unsigned of 5 octets", "25 05 01 02 03 04 05", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"a bit string with 8 unused bits", "82 08 00", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"a bit string with no octets but unused bits", "81 03", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"a character set other than 0", "72 04 41", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidParameterType},
		{"a context tag where an application value is required", "19 55", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"an application tag above 12", "D1 00", func(r *Reader) error { _, err := r.Value(); return err }, RejectInvalidTag},
		{"another context tag than the required one", "19 55", func(r *Reader) error { _, err := r.CtxUnsigned(0); return err }, RejectInvalidTag},
		{"nothing where a context tag is required", "", func(r *Reader) error { _, err := r.CtxOID(0); return err }, RejectMissingParameter},
		{"an object identifier of 3 octets", "0B 00 00 05", func(r *Reader) error { _, err := r.CtxOID(0); return err }, RejectInvalidTag},
		{"a list that is never closed", "3E 44 42 90 99 9A", func(r *Reader) error { _, err := r.Enclosed(3); return err }, RejectMissingParameter},
		{"a list closed by another tag", "3E 4F", func(r *Reader) error { _, err := r.Enclosed(3); return err }, RejectInvalidTag},
	}
	for _, c := range cases {
		err := c.read(NewReader(unhex(t, c.wire)))
		if got, ok := err.(Reject); !ok || got != c.want {
			t.Errorf("%s: %v, want reject %d", c.name, err, c.want)
		}
	}
}
