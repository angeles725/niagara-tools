package bacnet

import (
	"bytes"
	"reflect"
	"testing"
)

func at(i uint32) *uint32 { return &i }

// message is anything that encodes itself as service data.
type message interface{ Bytes() []byte }

// The frames marked "annex F" are the service data of the encoding examples of
// Annex F of ANSI/ASHRAE 135 (F.3.5 ReadProperty, F.3.7 ReadPropertyMultiple,
// F.3.9 WriteProperty, F.4.3 Who-Is and I-Am), written here from memory of the
// standard; the others were built from the rules of clauses 20 and 21.
var messages = []struct {
	name   string
	value  message
	wire   string
	decode func([]byte) (message, error)
}{
	{"read-property analog-input 5 present-value (annex F)", ReadProperty{ObjectID{AnalogInput, 5}, PropertyRef{Property: PropPresentValue}},
		"0C 00 00 00 05 19 55", func(b []byte) (message, error) { return DecodeReadProperty(b) }},
	{"read-property object-list[0]", ReadProperty{ObjectID{Device, 1001}, PropertyRef{PropObjectList, at(0)}},
		"0C 02 00 03 E9 19 4C 29 00", func(b []byte) (message, error) { return DecodeReadProperty(b) }},
	{"read-property acknowledgement 72.3 (annex F)", ReadPropertyAck{ObjectID{AnalogInput, 5}, PropertyRef{Property: PropPresentValue}, []Value{Real(72.3)}},
		"0C 00 00 00 05 19 55 3E 44 42 90 99 9A 3F", func(b []byte) (message, error) { return DecodeReadPropertyAck(b) }},
	{"read-property acknowledgement of an array element", ReadPropertyAck{ObjectID{Device, 1001}, PropertyRef{PropObjectList, at(2)}, []Value{OID(ObjectID{BinaryValue, 7})}},
		"0C 02 00 03 E9 19 4C 29 02 3E C4 01 40 00 07 3F", func(b []byte) (message, error) { return DecodeReadPropertyAck(b) }},
	{"read-property-multiple present-value and reliability (annex F)",
		ReadPropertyMultiple{{ObjectID{AnalogInput, 16}, []PropertyRef{{Property: PropPresentValue}, {Property: PropReliability}}}},
		"0C 00 00 00 10 1E 09 55 09 67 1F", func(b []byte) (message, error) { return DecodeReadPropertyMultiple(b) }},
	{"read-property-multiple of two objects, one with an index",
		ReadPropertyMultiple{{ObjectID{Device, 1}, []PropertyRef{{PropObjectList, at(1)}}}, {ObjectID{BinaryInput, 2}, []PropertyRef{{Property: PropAll}}}},
		"0C 02 00 00 01 1E 09 4C 19 01 1F 0C 00 C0 00 02 1E 09 08 1F", func(b []byte) (message, error) { return DecodeReadPropertyMultiple(b) }},
	{"read-property-multiple acknowledgement (annex F)",
		ReadPropertyMultipleAck{{ObjectID{AnalogInput, 16}, []ReadResult{
			{Ref: PropertyRef{Property: PropPresentValue}, Values: []Value{Real(72.3)}},
			{Ref: PropertyRef{Property: PropReliability}, Values: []Value{Enum(0)}}}}},
		"0C 00 00 00 10 1E 29 55 4E 44 42 90 99 9A 4F 29 67 4E 91 00 4F 1F", func(b []byte) (message, error) { return DecodeReadPropertyMultipleAck(b) }},
	{"read-property-multiple acknowledgement with an error and an index",
		ReadPropertyMultipleAck{{ObjectID{AnalogInput, 16}, []ReadResult{
			{Ref: PropertyRef{PropStateText, at(3)}, Err: ErrUnknownProperty},
			{Ref: PropertyRef{PropPriorityArray, at(8)}, Values: []Value{Null()}}}}},
		"0C 00 00 00 10 1E 29 6E 39 03 5E 91 02 91 20 5F 29 57 39 08 4E 00 4F 1F", func(b []byte) (message, error) { return DecodeReadPropertyMultipleAck(b) }},
	{"write-property analog-value 1 present-value 180 (annex F)",
		WriteProperty{Object: ObjectID{AnalogValue, 1}, Ref: PropertyRef{Property: PropPresentValue}, Values: []Value{Real(180)}},
		"0C 00 80 00 01 19 55 3E 44 43 34 00 00 3F", func(b []byte) (message, error) { return DecodeWriteProperty(b) }},
	{"write-property with a priority", WriteProperty{Object: ObjectID{AnalogOutput, 3}, Ref: PropertyRef{Property: PropPresentValue}, Values: []Value{Real(55)}, HasPriority: true, Priority: 8},
		"0C 00 40 00 03 19 55 3E 44 42 5C 00 00 3F 49 08", func(b []byte) (message, error) { return DecodeWriteProperty(b) }},
	{"write-property that relinquishes", WriteProperty{Object: ObjectID{BinaryOutput, 1}, Ref: PropertyRef{Property: PropPresentValue}, Values: []Value{Null()}, HasPriority: true, Priority: 16},
		"0C 01 00 00 01 19 55 3E 00 3F 49 10", func(b []byte) (message, error) { return DecodeWriteProperty(b) }},
	{"subscribe-cov, confirmed, with no end", SubscribeCOV{Process: 18, Object: ObjectID{AnalogInput, 10}, Confirmed: true},
		"09 12 1C 00 00 00 0A 29 01 39 00", func(b []byte) (message, error) { return DecodeSubscribeCOV(b) }},
	{"subscribe-cov, unconfirmed, for 300 s", SubscribeCOV{Process: 1, Object: ObjectID{BinaryInput, 2}, Lifetime: 300},
		"09 01 1C 00 C0 00 02 29 00 3A 01 2C", func(b []byte) (message, error) { return DecodeSubscribeCOV(b) }},
	{"subscribe-cov cancellation", SubscribeCOV{Process: 18, Object: ObjectID{AnalogInput, 10}, Cancel: true},
		"09 12 1C 00 00 00 0A", func(b []byte) (message, error) { return DecodeSubscribeCOV(b) }},
	{"cov notification", COVNotification{Process: 18, Device: ObjectID{Device, 4}, Object: ObjectID{AnalogInput, 10}, Remaining: 0,
		Values: []PropertyValue{{PropertyRef{Property: PropPresentValue}, []Value{Real(65)}}, {PropertyRef{Property: PropStatusFlags}, []Value{Bits(false, false, false, false)}}}},
		"09 12 1C 02 00 00 04 2C 00 00 00 0A 39 00 4E 09 55 2E 44 42 82 00 00 2F 09 6F 2E 82 04 00 2F 4F",
		func(b []byte) (message, error) { return DecodeCOVNotification(b) }},
	{"who-is with no range", WhoIs{}, "", func(b []byte) (message, error) { return DecodeWhoIs(b) }},
	{"who-is device 3 (annex F)", WhoIs{Ranged: true, Low: 3, High: 3}, "09 03 19 03", func(b []byte) (message, error) { return DecodeWhoIs(b) }},
	{"i-am device 3 (annex F)", IAm{Device: ObjectID{Device, 3}, MaxAPDU: 1024, Segmentation: SegmentationNone, Vendor: 99},
		"C4 02 00 00 03 22 04 00 91 03 21 63", func(b []byte) (message, error) { return DecodeIAm(b) }},
	{"who-has by name", WhoHas{ByName: true, Name: "OATemp"}, "3D 07 00 4F 41 54 65 6D 70", func(b []byte) (message, error) { return DecodeWhoHas(b) }},
	{"who-has by identifier in a range", WhoHas{Ranged: true, Low: 1, High: 9, Object: ObjectID{AnalogInput, 3}},
		"09 01 19 09 2C 00 00 00 03", func(b []byte) (message, error) { return DecodeWhoHas(b) }},
	{"i-have", IHave{Device: ObjectID{Device, 8}, Object: ObjectID{AnalogInput, 3}, Name: "OATemp"},
		"C4 02 00 00 08 C4 00 00 00 03 75 07 00 4F 41 54 65 6D 70", func(b []byte) (message, error) { return DecodeIHave(b) }},
}

func TestServiceDataEncodesAndDecodes(t *testing.T) {
	t.Parallel()
	for _, c := range messages {
		wire := unhex(t, c.wire)
		if got := c.value.Bytes(); !bytes.Equal(got, wire) {
			t.Errorf("%s: encoded % X\n\twant % X", c.name, got, wire)
		}
		got, err := c.decode(wire)
		if err != nil || !reflect.DeepEqual(got, c.value) {
			t.Errorf("%s: decoded %+v, %v\n\twant %+v", c.name, got, err, c.value)
		}
	}
}

func TestWholeFramesOfTheStandard(t *testing.T) {
	t.Parallel()
	// Annex F.3.5, F.3.9 and F.4.3: the APDUs as the standard prints them.
	for name, c := range map[string]struct {
		built []byte
		wire  string
	}{
		"read-property request":  {ConfirmedRequest(50, 1, ServiceReadProperty, ReadProperty{ObjectID{AnalogInput, 5}, PropertyRef{Property: PropPresentValue}}.Bytes()), "00 00 01 0C 0C 00 00 00 05 19 55"},
		"read-property ack":      {ComplexACK(1, ServiceReadProperty, ReadPropertyAck{ObjectID{AnalogInput, 5}, PropertyRef{Property: PropPresentValue}, []Value{Real(72.3)}}.Bytes()), "30 01 0C 0C 00 00 00 05 19 55 3E 44 42 90 99 9A 3F"},
		"write-property request": {ConfirmedRequest(1024, 89, ServiceWriteProperty, WriteProperty{Object: ObjectID{AnalogValue, 1}, Ref: PropertyRef{Property: PropPresentValue}, Values: []Value{Real(180)}}.Bytes()), "00 04 59 0F 0C 00 80 00 01 19 55 3E 44 43 34 00 00 3F"},
		"write-property ack":     {SimpleACK(89, ServiceWriteProperty), "20 59 0F"},
		"who-is":                 {UnconfirmedRequest(ServiceWhoIs, WhoIs{Ranged: true, Low: 3, High: 3}.Bytes()), "10 08 09 03 19 03"},
		"i-am":                   {UnconfirmedRequest(ServiceIAm, IAm{ObjectID{Device, 3}, 1024, SegmentationNone, 99}.Bytes()), "10 00 C4 02 00 00 03 22 04 00 91 03 21 63"},
	} {
		if want := unhex(t, c.wire); !bytes.Equal(c.built, want) {
			t.Errorf("%s: built % X\n\twant % X", name, c.built, want)
		}
	}
}

func TestServiceDataThatIsNotWellFormedIsRejected(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		wire   string
		decode func([]byte) error
		want   Reject
	}{
		{"read-property with no property", "0C 00 00 00 05", func(b []byte) error { _, err := DecodeReadProperty(b); return err }, RejectMissingParameter},
		{"read-property with no object", "19 55", func(b []byte) error { _, err := DecodeReadProperty(b); return err }, RejectInvalidTag},
		{"read-property with something after it", "0C 00 00 00 05 19 55 29 00 39 00", func(b []byte) error { _, err := DecodeReadProperty(b); return err }, RejectTooManyArguments},
		{"read-property-multiple with nothing", "", func(b []byte) error { _, err := DecodeReadPropertyMultiple(b); return err }, RejectMissingParameter},
		{"read-property-multiple with no property", "0C 00 00 00 10 1E 1F", func(b []byte) error { _, err := DecodeReadPropertyMultiple(b); return err }, RejectMissingParameter},
		{"read-property-multiple never closed", "0C 00 00 00 10 1E 09 55", func(b []byte) error { _, err := DecodeReadPropertyMultiple(b); return err }, RejectMissingParameter},
		{"write-property with no value", "0C 00 80 00 01 19 55", func(b []byte) error { _, err := DecodeWriteProperty(b); return err }, RejectMissingParameter},
		{"write-property with a priority of 2 octets cut short", "0C 00 80 00 01 19 55 3E 00 3F 4A 08", func(b []byte) error { _, err := DecodeWriteProperty(b); return err }, RejectInvalidTag},
		{"subscribe-cov with no object", "09 12", func(b []byte) error { _, err := DecodeSubscribeCOV(b); return err }, RejectMissingParameter},
		{"who-is with half a range", "09 03", func(b []byte) error { _, err := DecodeWhoIs(b); return err }, RejectMissingParameter},
		{"who-has with no object", "09 01 19 09", func(b []byte) error { _, err := DecodeWhoHas(b); return err }, RejectMissingParameter},
	}
	for _, c := range cases {
		err := c.decode(unhex(t, c.wire))
		if got, ok := err.(Reject); !ok || got != c.want {
			t.Errorf("%s: %v, want reject %d", c.name, err, c.want)
		}
	}
}
