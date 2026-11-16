package bacserver

import (
	"fmt"
	"math"
	"net/netip"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// subKey identifies a subscription as the standard does: the subscriber, its
// process identifier and the object.
type subKey struct {
	from    netip.AddrPort
	process uint32
	object  bacnet.ObjectID
}

// subscription is one COV subscription.
type subscription struct {
	confirmed bool
	ends      bool
	expires   time.Duration
	route     bacnet.NPDU    // how to reach a subscriber behind a router
	last      []bacnet.Value // present-value and status-flags as last notified; nil forces a notification
}

// prepareSubscribe decodes SubscribeCOV. A device whose scenario does not
// enable COV answers an error; one that does notifies at once and then on
// every change, until the lifetime ends or the subscription is cancelled.
func prepareSubscribe(x *exchange, data []byte) (call, error) {
	q, err := bacnet.DecodeSubscribeCOV(data)
	if err != nil {
		return call{}, err
	}
	x.entry.Object = q.Object.String()
	kind, life := "unconfirmed", "no end"
	if q.Confirmed {
		kind = "confirmed"
	}
	if q.Lifetime > 0 {
		life = fmt.Sprintf("%d s", q.Lifetime)
	}
	x.entry.Detail = fmt.Sprintf("process %d, %s, %s", q.Process, kind, life)
	if q.Cancel {
		x.entry.Detail = fmt.Sprintf("process %d, cancel", q.Process)
	}
	return call{run: func(x *exchange) ([]byte, *bacnet.Error) {
		s, key := x.s, subKey{from: x.from, process: q.Process, object: q.Object}
		switch {
		case !s.device.Spec().COV:
			return nil, bacnet.ErrCOVNotSupported
		case q.Cancel:
			s.mu.Lock()
			delete(s.subs, key)
			s.mu.Unlock()
			return nil, nil
		case !s.device.Has(q.Object):
			return nil, bacnet.ErrUnknownObject
		case s.device.Point(q.Object) == "":
			return nil, bacnet.ErrCOVNotSupported // the Device object has no present-value
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if _, renewal := s.subs[key]; !renewal && len(s.subs) >= s.opts.MaxSubscriptions {
			return nil, bacnet.ErrNoSubscriptionRoom
		}
		s.subs[key] = &subscription{confirmed: q.Confirmed, ends: q.Lifetime > 0,
			expires: x.elapsed + time.Duration(q.Lifetime)*time.Second, route: x.npdu.Reply(nil)}
		return nil, nil
	}, after: (*Server).CheckCOV}, nil
}

// changed reports whether values differ from the last notified ones by enough:
// an analog present-value by its COV increment, anything else at all.
func (s *Server) changed(object bacnet.ObjectID, last, now []bacnet.Value) bool {
	if last == nil || len(last) != len(now) || !last[1].Equal(now[1]) {
		return true
	}
	if increment, analog := s.device.COVIncrement(object); analog {
		diff := math.Abs(float64(now[0].Real) - float64(last[0].Real))
		return diff > 0 && diff >= increment
	}
	return !last[0].Equal(now[0])
}

// notice is one notification to send.
type notice struct {
	key    subKey
	sub    subscription
	invoke byte
	left   uint32
}

// CheckCOV sends a notification for every subscription whose value changed
// since its last one, and drops the subscriptions whose lifetime ended. It
// runs after each write and each new subscription, on the server's timer, and
// whenever the control port sets a point.
func (s *Server) CheckCOV() {
	now := s.opts.Elapsed()
	offline := s.device.Model().Offline(now)
	var due []notice
	s.mu.Lock()
	for key, sub := range s.subs {
		if sub.ends && now >= sub.expires {
			delete(s.subs, key)
			continue
		}
		if offline {
			continue
		}
		pv, err1 := s.device.Read(key.object, bacnet.PropertyRef{Property: bacnet.PropPresentValue}, now)
		flags, err2 := s.device.Read(key.object, bacnet.PropertyRef{Property: bacnet.PropStatusFlags}, now)
		if err1 != nil || err2 != nil {
			continue
		}
		values := []bacnet.Value{pv[0], flags[0]}
		if !s.changed(key.object, sub.last, values) {
			continue
		}
		sub.last = values
		n := notice{key: key, sub: *sub}
		if sub.ends {
			n.left = uint32((sub.expires - now + time.Second - 1) / time.Second)
		}
		if sub.confirmed {
			s.invoke++
			n.invoke = s.invoke
		}
		due = append(due, n)
	}
	s.mu.Unlock()
	for _, n := range due {
		s.notify(n, now)
	}
}

// notify sends one COV notification and logs it. A confirmed one is sent
// once: its acknowledgement is not awaited and it is not retried.
func (s *Server) notify(n notice, now time.Duration) {
	data := bacnet.COVNotification{Process: n.key.process, Device: s.device.ID(), Object: n.key.object, Remaining: n.left,
		Values: []bacnet.PropertyValue{
			{Ref: bacnet.PropertyRef{Property: bacnet.PropPresentValue}, Values: n.sub.last[:1]},
			{Ref: bacnet.PropertyRef{Property: bacnet.PropStatusFlags}, Values: n.sub.last[1:]},
		}}.Bytes()
	entry := s.newEntry(n.key.from, bacnet.OriginalUnicast, now)
	route := n.sub.route
	if n.sub.confirmed {
		invoke := int(n.invoke)
		entry.InvokeID, entry.PDU, entry.Service = &invoke, bacnet.PDUName(bacnet.PDUConfirmed), bacnet.ServiceName(true, bacnet.ServiceConfirmedCOVNotification)
		route.ExpectingReply = true
		route.APDU = bacnet.ConfirmedRequest(bacnet.MaxAPDU, n.invoke, bacnet.ServiceConfirmedCOVNotification, data)
	} else {
		entry.PDU, entry.Service = bacnet.PDUName(bacnet.PDUUnconfirmed), bacnet.ServiceName(false, bacnet.ServiceUnconfirmedCOVNotification)
		route.APDU = bacnet.UnconfirmedRequest(bacnet.ServiceUnconfirmedCOVNotification, data)
	}
	entry.Object, entry.Read, entry.Result = n.key.object.String(), texts(n.sub.last), reqlog.ResultSent
	entry.To = []string{n.key.from.String()}
	s.opts.Log(entry)
	s.send(n.key.from, bacnet.BVLL{Function: bacnet.OriginalUnicast, Data: route.Bytes()})
}
