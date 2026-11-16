package bacnet

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAPDUHeadersEncodeAndDecode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		wire []byte
		want APDU
	}{
		{"a confirmed request", ConfirmedRequest(1476, 7, ServiceReadProperty, []byte{0xAA}),
			APDU{Type: PDUConfirmed, MaxAPDU: 1476, InvokeID: 7, Service: ServiceReadProperty, Data: []byte{0xAA}}},
		{"a confirmed request of a small client", ConfirmedRequest(50, 1, ServiceReadProperty, nil),
			APDU{Type: PDUConfirmed, MaxAPDU: 50, InvokeID: 1, Service: ServiceReadProperty}},
		{"an unconfirmed request", UnconfirmedRequest(ServiceWhoIs, nil), APDU{Type: PDUUnconfirmed, Service: ServiceWhoIs}},
		{"a simple acknowledgement", SimpleACK(89, ServiceWriteProperty), APDU{Type: PDUSimpleACK, InvokeID: 89, Service: ServiceWriteProperty}},
		{"a complex acknowledgement", ComplexACK(1, ServiceReadProperty, []byte{1, 2}),
			APDU{Type: PDUComplexACK, InvokeID: 1, Service: ServiceReadProperty, Data: []byte{1, 2}}},
		{"an error", ErrorPDU(3, ServiceReadProperty, *ErrUnknownObject),
			APDU{Type: PDUError, InvokeID: 3, Service: ServiceReadProperty, Err: *ErrUnknownObject, Data: []byte{0x91, 1, 0x91, 31}}},
		{"a reject", RejectPDU(4, RejectUnrecognizedService), APDU{Type: PDUReject, InvokeID: 4, Reason: 9}},
		{"an abort by the server", AbortPDU(5, AbortSegmentationNotSupported), APDU{Type: PDUAbort, InvokeID: 5, Reason: 4, Server: true}},
	}
	for _, c := range cases {
		got, ok := ParseAPDU(c.wire)
		if !ok || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s (% X): decoded %+v, %v; want %+v", c.name, c.wire, got, ok, c.want)
		}
	}
	for name, wire := range map[string]string{
		"an error":  "50 03 0C 91 01 91 1F",
		"a reject":  "60 04 09",
		"an abort":  "71 05 04",
		"a request": "00 05 07 0C AA",
	} {
		var built []byte
		switch name {
		case "an error":
			built = ErrorPDU(3, ServiceReadProperty, *ErrUnknownObject)
		case "a reject":
			built = RejectPDU(4, RejectUnrecognizedService)
		case "an abort":
			built = AbortPDU(5, AbortSegmentationNotSupported)
		default:
			built = ConfirmedRequest(1476, 7, ServiceReadProperty, []byte{0xAA})
		}
		if want := unhex(t, wire); !bytes.Equal(built, want) {
			t.Errorf("%s: encoded % X, want % X", name, built, want)
		}
	}
	segmented, ok := ParseAPDU(unhex(t, "0C 05 07 00 04 0C AA"))
	if !ok || !segmented.Segmented || !segmented.MoreFollows || segmented.InvokeID != 7 || segmented.Service != ServiceReadProperty {
		t.Errorf("a segmented request: %+v, %v", segmented, ok)
	}
	for name, wire := range map[string]string{
		"nothing": "", "a confirmed request with no service": "00 05 07", "an unconfirmed request with no service": "10",
		"a reject with no reason": "60 04", "a simple acknowledgement cut short": "20 01", "an error with no class": "50 03 0C",
		"a segmented request with no window": "08 05 07 00",
	} {
		if got, ok := ParseAPDU(unhex(t, wire)); ok {
			t.Errorf("%s was accepted as %+v", name, got)
		}
	}
	for code, size := range map[int]int{50: 0, 128: 1, 206: 2, 480: 3, 1024: 4, 1476: 5} {
		if got := ConfirmedRequest(code, 0, 0, nil)[1]; int(got) != size {
			t.Errorf("max APDU %d is coded %d, want %d", code, got, size)
		}
	}
}
