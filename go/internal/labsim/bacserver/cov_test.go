package bacserver

import (
	"net"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

func (s *sim) subscribe(c *net.UDPConn, q bacnet.SubscribeCOV) bacnet.APDU {
	s.t.Helper()
	return s.ask(c, 30, bacnet.ServiceSubscribeCOV, q.Bytes())
}

// notification reads the next frame as a COV notification.
func (s *sim) notification(c *net.UDPConn, confirmed bool) (bacnet.COVNotification, bacnet.APDU) {
	s.t.Helper()
	p := s.recv(c)
	n, err := bacnet.DecodeCOVNotification(p.apdu.Data)
	wantType, wantService := byte(bacnet.PDUUnconfirmed), byte(bacnet.ServiceUnconfirmedCOVNotification)
	if confirmed {
		wantType, wantService = bacnet.PDUConfirmed, bacnet.ServiceConfirmedCOVNotification
	}
	if err != nil || p.apdu.Type != wantType || p.apdu.Service != wantService || p.bvll.Function != bacnet.OriginalUnicast {
		s.t.Fatalf("%+v (%v) is not the expected COV notification", p.apdu, err)
	}
	return n, p.apdu
}

func TestSubscribeCOVIsAnErrorUnlessTheDeviceEnablesIt(t *testing.T) {
	t.Parallel()
	s := start(t, bench(false, 0), nil)
	c := s.client()
	a := s.subscribe(c, bacnet.SubscribeCOV{Process: 1, Object: supply, Lifetime: 60})
	if a.Type != bacnet.PDUError || a.Service != bacnet.ServiceSubscribeCOV || a.Err != *bacnet.ErrCOVNotSupported {
		t.Errorf("%+v, want error services / optional-functionality-not-supported", a)
	}
	s.server.CheckCOV()
	s.settle(c)
}

func TestASubscriptionGetsANotificationWhenTheValueChanges(t *testing.T) {
	t.Parallel()
	s := start(t, bench(true, 0), nil)
	c := s.client()
	if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 18, Object: supply, Lifetime: 60}); a.Type != bacnet.PDUSimpleACK || a.Service != bacnet.ServiceSubscribeCOV {
		t.Fatalf("subscribe: %+v", a)
	}
	if e := s.logged()[0]; e.Service != "subscribe-cov" || e.Object != "analog-input,1" || e.Detail != "process 18, unconfirmed, 60 s" || e.Result != reqlog.ResultOK {
		t.Errorf("log entry %+v", e)
	}
	first, _ := s.notification(c, false)
	if first.Process != 18 || first.Device != deviceID || first.Object != supply || first.Remaining != 60 || len(first.Values) != 2 ||
		first.Values[0].Ref.Property != bacnet.PropPresentValue || first.Values[0].Values[0].Real != 18.5 ||
		first.Values[1].Ref.Property != bacnet.PropStatusFlags || len(first.Values[1].Values[0].Bits) != 4 {
		t.Errorf("the notification a subscription starts with: %+v", first)
	}
	if got := s.read(c, deviceID, ref(bacnet.PropServicesSupported)); !got[0].Bits[bacnet.SupportedSubscribeCOV] ||
		!got[0].Bits[bacnet.SupportedConfirmedCOVNotification] || !got[0].Bits[bacnet.SupportedUnconfirmedCOVNotification] {
		t.Errorf("protocol-services-supported of a device with COV = %v", got[0].Bits)
	}
	s.device.Model().Pin("supply_temp", 18.9) // below the increment of 0.5
	s.server.CheckCOV()
	s.settle(c)
	s.advance(15 * time.Second)
	s.device.Model().Pin("supply_temp", 19.25)
	s.server.CheckCOV() // 10-20 s: the device is offline and sends nothing
	s.advance(10 * time.Second)
	s.settle(c)
	s.server.CheckCOV()
	changed, _ := s.notification(c, false)
	if changed.Values[0].Values[0].Real != 19.25 || changed.Remaining != 35 {
		t.Errorf("after the change: %+v, want 19.25 with 35 s left", changed)
	}
	if e := s.last(); e.Service != "unconfirmed-cov-notification" || e.Result != reqlog.ResultSent || e.Object != "analog-input,1" || len(e.To) != 1 || e.To[0] != local(c).String() {
		t.Errorf("log entry %+v", e)
	}
	s.server.CheckCOV() // nothing changed since
	s.settle(c)
	s.advance(36 * time.Second) // the lifetime ran out
	s.device.Model().Pin("supply_temp", 30)
	s.server.CheckCOV()
	s.settle(c)
	if got := s.server.Stats().Subscriptions; got != 0 {
		t.Errorf("Stats().Subscriptions = %d after the lifetime ran out", got)
	}
}

func TestAConfirmedSubscriptionOfACommandedOutputAndItsCancellation(t *testing.T) {
	t.Parallel()
	s := start(t, bench(true, 0), nil)
	c := s.client()
	if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 7, Object: command, Confirmed: true}); a.Type != bacnet.PDUSimpleACK {
		t.Fatalf("subscribe: %+v", a)
	}
	first, apdu := s.notification(c, true)
	if first.Remaining != 0 || first.Values[0].Values[0].Uint != 0 {
		t.Errorf("the first notification: %+v, want inactive and no end", first)
	}
	// The client acknowledges; the simulator takes it without answering.
	s.unicast(c, bacnet.SimpleACK(apdu.InvokeID, bacnet.ServiceConfirmedCOVNotification))
	// A write is followed by its notification with no CheckCOV from outside.
	if a := s.write(c, command, 8, bacnet.Enum(1)); a.Type != bacnet.PDUSimpleACK {
		t.Fatalf("write: %+v", a)
	}
	second, next := s.notification(c, true)
	if second.Values[0].Values[0].Uint != 1 || next.InvokeID == apdu.InvokeID {
		t.Errorf("after the command: %+v (invoke id %d then %d), want active under a new invoke id", second, apdu.InvokeID, next.InvokeID)
	}
	if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 7, Object: command, Cancel: true}); a.Type != bacnet.PDUSimpleACK {
		t.Fatalf("cancel: %+v", a)
	}
	s.write(c, command, 8, bacnet.Null())
	s.settle(c)
	if got := s.server.Stats().Subscriptions; got != 0 {
		t.Errorf("Stats().Subscriptions = %d after the cancellation", got)
	}
}

func TestSubscriptionsAreBoundedAndRefusedForWhatHasNoValue(t *testing.T) {
	t.Parallel()
	s := start(t, bench(true, 0), func(o *Options) { o.MaxSubscriptions = 2 })
	c := s.client()
	for name, c2 := range map[string]struct {
		object bacnet.ObjectID
		want   *bacnet.Error
	}{
		"an unknown object": {bacnet.ObjectID{Type: bacnet.AnalogInput, Instance: 99}, bacnet.ErrUnknownObject},
		"the device object": {deviceID, bacnet.ErrCOVNotSupported},
	} {
		if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 1, Object: c2.object}); a.Type != bacnet.PDUError || a.Err != *c2.want {
			t.Errorf("%s: %+v, want %v", name, a, c2.want)
		}
	}
	for _, object := range []bacnet.ObjectID{supply, status} {
		s.subscribe(c, bacnet.SubscribeCOV{Process: 1, Object: object})
		s.notification(c, false)
	}
	if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 1, Object: damper}); a.Type != bacnet.PDUError || a.Err != *bacnet.ErrNoSubscriptionRoom {
		t.Errorf("a third subscription: %+v, want error resources / no-space-to-add-list-element", a)
	}
	// Subscribing again to the same object renews it and takes no room.
	if a := s.subscribe(c, bacnet.SubscribeCOV{Process: 1, Object: supply, Lifetime: 30}); a.Type != bacnet.PDUSimpleACK {
		t.Errorf("a renewal: %+v", a)
	}
	if n, _ := s.notification(c, false); n.Remaining != 30 {
		t.Errorf("the renewal's notification has %d s left, want 30", n.Remaining)
	}
	if got := s.server.Stats().Subscriptions; got != 2 {
		t.Errorf("Stats().Subscriptions = %d, want 2", got)
	}
}
