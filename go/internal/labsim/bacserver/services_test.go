package bacserver

import (
	"fmt"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

func TestWhoIsIsAnsweredWithIAmToTheAsker(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	for _, q := range []bacnet.WhoIs{{}, {Ranged: true, Low: 1000, High: 1001}} {
		name := fmt.Sprintf("%+v", q)
		s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceWhoIs, q.Bytes()))
		p := s.recv(c)
		iam, err := bacnet.DecodeIAm(p.apdu.Data)
		want := bacnet.IAm{Device: deviceID, MaxAPDU: 1476, Segmentation: bacnet.SegmentationNone, Vendor: 999}
		if p.bvll.Function != bacnet.OriginalUnicast || p.apdu.Type != bacnet.PDUUnconfirmed || p.apdu.Service != bacnet.ServiceIAm || err != nil || iam != want {
			t.Errorf("%s: %+v, %+v, %v; want a unicast I-Am %+v", name, p, iam, err, want)
		}
	}
	if e := s.last(); e.Service != "who-is" || e.Result != reqlog.ResultOK || e.Protocol != "bacnet-ip" || e.Device != "unit-a" ||
		e.Link != "original-unicast-npdu" || len(e.To) != 1 || e.To[0] != local(c).String() || e.Detail != "1000-1001" {
		t.Errorf("log entry %+v", e)
	}
	s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceWhoIs, bacnet.WhoIs{Ranged: true, Low: 1, High: 1000}.Bytes()))
	s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceWhoIs, []byte{0x09})) // half a tag
	s.settle(c)
	entries := s.logged()
	if e := entries[len(entries)-3]; e.Result != reqlog.ResultIgnored || e.Reason != "not_in_range" {
		t.Errorf("a Who-Is for other devices: %+v", e)
	}
	if e := entries[len(entries)-2]; e.Result != reqlog.ResultIgnored || e.Reason != "invalid-tag" {
		t.Errorf("a Who-Is that is not well formed: %+v", e)
	}
}

func TestWhoHasIsAnsweredWithIHave(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	for name, q := range map[string]bacnet.WhoHas{
		"by name":       {ByName: true, Name: "damper"},
		"by identifier": {Ranged: true, Low: 1001, High: 1001, Object: damper},
	} {
		s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceWhoHas, q.Bytes()))
		p := s.recv(c)
		have, err := bacnet.DecodeIHave(p.apdu.Data)
		if p.apdu.Service != bacnet.ServiceIHave || err != nil || have != (bacnet.IHave{Device: deviceID, Object: damper, Name: "damper"}) {
			t.Errorf("%s: %+v, %v", name, have, err)
		}
	}
	for _, q := range []bacnet.WhoHas{{ByName: true, Name: "absent"}, {Object: bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 99}},
		{Ranged: true, Low: 1, High: 9, Object: damper}} {
		s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceWhoHas, q.Bytes()))
	}
	s.settle(c)
}

func TestTimeSynchronizationIsAcceptedAndLogged(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	stamp := bacnet.Value{Kind: bacnet.KindTime, Raw: []byte{10, 30, 0, 0}}.Append(bacnet.Value{Kind: bacnet.KindDate, Raw: []byte{126, 11, 15, 7}}.Append(nil))
	s.unicast(c, bacnet.UnconfirmedRequest(bacnet.ServiceTimeSynchronization, stamp))
	s.unicast(c, bacnet.UnconfirmedRequest(3, nil)) // an event notification: not a service of the simulator
	s.settle(c)
	entries := s.logged()
	if e := entries[len(entries)-3]; e.Service != "time-synchronization" || e.Result != reqlog.ResultOK || e.Detail != "2026-11-15 10:30:00.00" {
		t.Errorf("time synchronization: %+v", e)
	}
	if e := entries[len(entries)-2]; e.Service != "unconfirmed-event-notification" || e.Result != reqlog.ResultIgnored || e.Reason != "unsupported_service" {
		t.Errorf("another unconfirmed service: %+v", e)
	}
}

func TestReadPropertyAnswersValuesAndErrors(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	if got := s.read(c, supply, ref(bacnet.PropPresentValue)); len(got) != 1 || got[0].Real != 18.5 {
		t.Errorf("present-value = %+v", got)
	}
	if e := s.last(); e.Service != "read-property" || e.Object != "analog-input,1" || e.Property != "present-value" || e.Result != reqlog.ResultOK ||
		e.InvokeID == nil || *e.InvokeID != 1 || len(e.Read) != 1 || e.Read[0] != "18.5" || e.PDU != "confirmed-request" {
		t.Errorf("log entry %+v", e)
	}
	if got := s.read(c, deviceID, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(0)}); got[0].Uint != 9 {
		t.Errorf("object-list[0] = %+v, want 9", got)
	}
	if got := s.read(c, deviceID, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(3)}); got[0].Object != damper {
		t.Errorf("object-list[3] = %+v", got)
	}
	if got := s.read(c, deviceID, ref(bacnet.PropObjectList)); len(got) != 9 {
		t.Errorf("the whole object-list has %d elements", len(got))
	}
	if got := s.read(c, deviceID, ref(bacnet.PropServicesSupported)); len(got[0].Bits) != 40 || !got[0].Bits[bacnet.SupportedReadProperty] ||
		!got[0].Bits[bacnet.SupportedReadPropertyMultiple] || !got[0].Bits[bacnet.SupportedWriteProperty] || !got[0].Bits[bacnet.SupportedWhoIs] ||
		!got[0].Bits[bacnet.SupportedIAm] || !got[0].Bits[bacnet.SupportedWhoHas] || !got[0].Bits[bacnet.SupportedTimeSynchronization] ||
		got[0].Bits[bacnet.SupportedSubscribeCOV] || got[0].Bits[bacnet.SupportedConfirmedCOVNotification] || got[0].Bits[16] {
		t.Errorf("protocol-services-supported = %v", got[0].Bits)
	}
	for name, c2 := range map[string]struct {
		object bacnet.ObjectID
		ref    bacnet.PropertyRef
		want   *bacnet.Error
	}{
		"an unknown object":   {bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 99}, ref(bacnet.PropPresentValue), bacnet.ErrUnknownObject},
		"an unknown property": {supply, ref(bacnet.PropStateText), bacnet.ErrUnknownProperty},
		"a set of properties": {supply, ref(bacnet.PropAll), bacnet.ErrUnknownProperty},
		"a bad array index":   {deviceID, bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(10)}, bacnet.ErrInvalidArrayIndex},
	} {
		a := s.ask(c, 9, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: c2.object, Ref: c2.ref}.Bytes())
		if a.Type != bacnet.PDUError || a.Service != bacnet.ServiceReadProperty || a.Err != *c2.want {
			t.Errorf("%s: %+v, want error %v", name, a, c2.want)
		}
		if e := s.last(); e.Result != reqlog.ResultError || e.ErrorClass != c2.want.ClassName() || e.ErrorCode != c2.want.CodeName() {
			t.Errorf("%s: log entry %+v", name, e)
		}
	}
}

func TestReadPropertyMultipleAnswersEveryReferenceWithItsValueOrItsError(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	q := bacnet.ReadPropertyMultiple{
		{Object: supply, Refs: []bacnet.PropertyRef{ref(bacnet.PropPresentValue), ref(bacnet.PropStateText), ref(bacnet.PropUnits)}},
		{Object: damper, Refs: []bacnet.PropertyRef{{Property: bacnet.PropPriorityArray, Index: at(17)}, {Property: bacnet.PropPriorityArray, Index: at(0)}}},
		{Object: bacnet.ObjectID{Type: bacnet.BinaryInput, Instance: 99}, Refs: []bacnet.PropertyRef{ref(bacnet.PropPresentValue), ref(bacnet.PropAll)}},
		{Object: status, Refs: []bacnet.PropertyRef{ref(bacnet.PropRequired)}},
	}
	a := s.ask(c, 3, bacnet.ServiceReadPropertyMultiple, q.Bytes())
	ack, err := bacnet.DecodeReadPropertyMultipleAck(a.Data)
	if a.Type != bacnet.PDUComplexACK || err != nil || len(ack) != 4 {
		t.Fatalf("%+v, %v", a, err)
	}
	first, second, third, fourth := ack[0].Results, ack[1].Results, ack[2].Results, ack[3].Results
	if len(first) != 3 || first[0].Values[0].Real != 18.5 || first[1].Err == nil || *first[1].Err != *bacnet.ErrUnknownProperty || first[2].Values[0].Uint != 62 {
		t.Errorf("the analog input: %+v", first)
	}
	if len(second) != 2 || second[0].Err == nil || *second[0].Err != *bacnet.ErrInvalidArrayIndex || *second[0].Ref.Index != 17 || second[1].Values[0].Uint != 16 {
		t.Errorf("the priority array: %+v", second)
	}
	if len(third) != 2 || *third[0].Err != *bacnet.ErrUnknownObject || third[1].Ref.Property != bacnet.PropAll || *third[1].Err != *bacnet.ErrUnknownObject {
		t.Errorf("the unknown object: %+v", third)
	}
	if len(fourth) != 8 || fourth[0].Ref.Property != bacnet.PropObjectIdentifier || fourth[7].Ref.Property != bacnet.PropPolarity {
		t.Errorf("REQUIRED of a binary input: %+v", fourth)
	}
	if e := s.last(); e.Service != "read-property-multiple" || e.Detail != "4 objects, 8 references" || e.Result != reqlog.ResultOK {
		t.Errorf("log entry %+v", e)
	}
	all := s.ask(c, 4, bacnet.ServiceReadPropertyMultiple, bacnet.ReadPropertyMultiple{{Object: damper, Refs: []bacnet.PropertyRef{ref(bacnet.PropAll)}}}.Bytes())
	ack, err = bacnet.DecodeReadPropertyMultipleAck(all.Data)
	if err != nil || len(ack) != 1 || len(ack[0].Results) != 13 {
		t.Errorf("ALL of the analog output: %+v, %v", ack, err)
	}
}

func TestWritePropertyCommandsWithAPriorityAndRelinquishes(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	pv := ref(bacnet.PropPresentValue)
	if a := s.write(c, damper, 8, bacnet.Real(55)); a.Type != bacnet.PDUSimpleACK || a.Service != bacnet.ServiceWriteProperty {
		t.Fatalf("a write at priority 8: %+v", a)
	}
	if e := s.last(); e.Service != "write-property" || e.Object != "analog-output,1" || e.Priority == nil || *e.Priority != 8 || e.Written != "55" || e.Result != reqlog.ResultOK {
		t.Errorf("log entry %+v", e)
	}
	if got := s.read(c, damper, pv); got[0].Real != 55 {
		t.Errorf("commanded = %+v", got)
	}
	if got := s.read(c, damper, bacnet.PropertyRef{Property: bacnet.PropPriorityArray, Index: at(8)}); got[0].Real != 55 {
		t.Errorf("priority-array[8] = %+v", got)
	}
	if a := s.write(c, damper, 8, bacnet.Null()); a.Type != bacnet.PDUSimpleACK {
		t.Fatalf("a relinquish: %+v", a)
	}
	if e := s.last(); e.Written != "null" {
		t.Errorf("a relinquish is logged as %+v", e)
	}
	if got := s.read(c, damper, pv); got[0].Real != 35 {
		t.Errorf("relinquished = %+v, want the relinquish default 35", got)
	}
	if a := s.write(c, setpoint, 0, bacnet.Real(19)); a.Type != bacnet.PDUSimpleACK {
		t.Fatalf("a plain write: %+v", a)
	}
	if v, _ := s.device.Model().Get("setpoint", 0); v != 19 {
		t.Errorf("the model point = %v after a plain write of 19", v)
	}
	for name, c2 := range map[string]struct {
		object bacnet.ObjectID
		value  bacnet.Value
		want   *bacnet.Error
	}{
		"an input":                   {supply, bacnet.Real(1), bacnet.ErrWriteAccessDenied},
		"a value out of range":       {damper, bacnet.Real(101), bacnet.ErrValueOutOfRange},
		"a value of another type":    {damper, bacnet.Unsigned(1), bacnet.ErrInvalidDataType},
		"an object that is not here": {bacnet.ObjectID{Type: bacnet.AnalogValue, Instance: 99}, bacnet.Real(1), bacnet.ErrUnknownObject},
	} {
		if a := s.write(c, c2.object, 8, c2.value); a.Type != bacnet.PDUError || a.Service != bacnet.ServiceWriteProperty || a.Err != *c2.want {
			t.Errorf("%s: %+v, want error %v", name, a, c2.want)
		}
	}
}

func TestWhatIsNotAServiceOfTheSimulatorIsRejectedOrAborted(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	if a := s.ask(c, 5, 20, nil); a.Type != bacnet.PDUReject || bacnet.Reject(a.Reason) != bacnet.RejectUnrecognizedService {
		t.Errorf("reinitialize-device: %+v, want a reject (unrecognized service)", a)
	}
	if e := s.last(); e.Service != "reinitialize-device" || e.Result != reqlog.ResultReject || e.Reason != "unrecognized-service" {
		t.Errorf("log entry %+v", e)
	}
	if a := s.ask(c, 6, bacnet.ServiceReadProperty, []byte{0x0C, 0, 0, 0, 5}); a.Type != bacnet.PDUReject || bacnet.Reject(a.Reason) != bacnet.RejectMissingParameter {
		t.Errorf("a read-property with no property: %+v, want a reject (missing parameter)", a)
	}
	s.unicast(c, []byte{0x0C, 0x05, 7, 0, 4, bacnet.ServiceReadProperty, 0x0C}) // the first segment of several
	if p := s.recv(c); p.apdu.Type != bacnet.PDUAbort || !p.apdu.Server || p.apdu.InvokeID != 7 || bacnet.Abort(p.apdu.Reason) != bacnet.AbortSegmentationNotSupported {
		t.Errorf("a segmented request: %+v, want an abort (segmentation not supported)", p.apdu)
	}
	// The object list is 9 identifiers of 5 octets: it does not fit the 50
	// octets of a small client, which then reads it by index.
	whole := bacnet.ReadProperty{Object: deviceID, Ref: ref(bacnet.PropObjectList)}.Bytes()
	s.unicast(c, bacnet.ConfirmedRequest(50, 8, bacnet.ServiceReadProperty, whole))
	if p := s.recv(c); p.apdu.Type != bacnet.PDUAbort || !p.apdu.Server || p.apdu.InvokeID != 8 || bacnet.Abort(p.apdu.Reason) != bacnet.AbortSegmentationNotSupported {
		t.Errorf("a response over the client's max APDU: %+v, want an abort (segmentation not supported)", p.apdu)
	}
	if e := s.last(); e.Result != reqlog.ResultAbort || e.Reason != "segmentation-not-supported" {
		t.Errorf("log entry %+v", e)
	}
	indexed := bacnet.ReadProperty{Object: deviceID, Ref: bacnet.PropertyRef{Property: bacnet.PropObjectList, Index: at(7)}}.Bytes()
	s.unicast(c, bacnet.ConfirmedRequest(50, 9, bacnet.ServiceReadProperty, indexed))
	if p := s.recv(c); p.apdu.Type != bacnet.PDUComplexACK || p.apdu.InvokeID != 9 {
		t.Errorf("the indexed read of the same list: %+v", p.apdu)
	}
}

func TestAResponseOverTheDevicesOwnMaxAPDUIsAborted(t *testing.T) {
	t.Parallel()
	s := start(t, benchAPDU(false, 0, 50), nil)
	c := s.client()
	a := s.ask(c, 1, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: deviceID, Ref: ref(bacnet.PropObjectList)}.Bytes())
	if a.Type != bacnet.PDUAbort || bacnet.Abort(a.Reason) != bacnet.AbortSegmentationNotSupported {
		t.Errorf("%+v, want an abort: the client accepts 1476 octets but the device sends 50 at most", a)
	}
}

func TestADuplicateRequestGetsTheSameAnswerWithoutBeingRunAgain(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	read := bacnet.ConfirmedRequest(bacnet.MaxAPDU, 40, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: fragile, Ref: ref(bacnet.PropPresentValue)}.Bytes())
	s.unicast(c, read)
	first := s.recv(c)
	s.unicast(c, read) // a retry: the point fails after one read, so running it again would be an error
	second := s.recv(c)
	if first.apdu.Type != bacnet.PDUComplexACK || second.apdu.Type != bacnet.PDUComplexACK || string(first.npdu.APDU) != string(second.npdu.APDU) {
		t.Errorf("first %+v, retry %+v; want the same acknowledgement", first.apdu, second.apdu)
	}
	if e := s.last(); !e.Duplicate || e.Result != reqlog.ResultOK {
		t.Errorf("the retry is logged as %+v", e)
	}
	if got := s.device.Model().Reads(); got != 1 {
		t.Errorf("%d reads were counted, want 1", got)
	}
	// Another request under the same invoke id is a new request.
	other := s.ask(c, 40, bacnet.ServiceReadProperty, bacnet.ReadProperty{Object: fragile, Ref: ref(bacnet.PropUnits)}.Bytes())
	if other.Type != bacnet.PDUError || other.Err != *bacnet.ErrOperationalProblem {
		t.Errorf("a new request under a used invoke id: %+v, want the failed point's error", other)
	}
}
