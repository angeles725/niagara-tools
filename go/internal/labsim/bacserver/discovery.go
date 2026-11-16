package bacserver

import (
	"fmt"
	"slices"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// unconfirmedServices are the unconfirmed services the simulator serves.
var unconfirmedServices map[byte]service

func init() {
	unconfirmedServices = map[byte]service{
		bacnet.ServiceWhoIs:                  {supported: []int{bacnet.SupportedWhoIs, bacnet.SupportedIAm}, handle: (*exchange).whoIs},
		bacnet.ServiceWhoHas:                 {supported: []int{bacnet.SupportedWhoHas, bacnet.SupportedIHave}, handle: (*exchange).whoHas},
		bacnet.ServiceTimeSynchronization:    {supported: []int{bacnet.SupportedTimeSynchronization}, handle: (*exchange).timeSync},
		bacnet.ServiceUTCTimeSynchronization: {supported: []int{bacnet.SupportedUTCTimeSynchronization}, handle: (*exchange).timeSync},
	}
}

// bad ignores a request that is not well formed, naming what a confirmed
// request would be rejected for.
func (x *exchange) bad(err error) {
	if reject, ok := err.(bacnet.Reject); ok {
		x.ignore(reject.String())
		return
	}
	x.ignore(reasonMalformed)
}

// whoIs answers Who-Is with I-Am when the device is in the range asked.
func (x *exchange) whoIs(data []byte) {
	q, err := bacnet.DecodeWhoIs(data)
	if err != nil {
		x.bad(err)
		return
	}
	if q.Ranged {
		x.entry.Detail = fmt.Sprintf("%d-%d", q.Low, q.High)
	}
	spec := x.s.device.Spec()
	if !q.Matches(spec.Instance) {
		x.ignore(reasonNotInRange)
		return
	}
	iam := bacnet.IAm{Device: x.s.device.ID(), MaxAPDU: uint32(spec.MaxAPDU), Segmentation: bacnet.SegmentationNone, Vendor: spec.VendorID}
	x.announce(bacnet.UnconfirmedRequest(bacnet.ServiceIAm, iam.Bytes()))
}

// whoHas answers Who-Has with I-Have when the device has the object, named or
// identified.
func (x *exchange) whoHas(data []byte) {
	q, err := bacnet.DecodeWhoHas(data)
	if err != nil {
		x.bad(err)
		return
	}
	device := x.s.device
	object, found := q.Object, device.Has(q.Object)
	x.entry.Detail = q.Object.String()
	if q.ByName {
		object, found = device.FindByName(q.Name)
		x.entry.Detail = q.Name
	}
	if q.Ranged && (device.ID().Instance < q.Low || device.ID().Instance > q.High) {
		x.ignore(reasonNotInRange)
		return
	}
	name, fault := device.Read(object, bacnet.PropertyRef{Property: bacnet.PropObjectName}, x.elapsed)
	if !found || fault != nil || len(name) != 1 {
		x.ignore(reasonNoObject)
		return
	}
	have := bacnet.IHave{Device: device.ID(), Object: object, Name: name[0].Text}
	x.announce(bacnet.UnconfirmedRequest(bacnet.ServiceIHave, have.Bytes()))
}

// timeSync accepts a time synchronization: the date and time go to the log
// and nowhere else, for the simulator's clock is its own.
func (x *exchange) timeSync(data []byte) {
	r := bacnet.NewReader(data)
	date, err1 := r.Value()
	clock, err2 := r.Value()
	if err1 != nil || err2 != nil || date.Kind != bacnet.KindDate || clock.Kind != bacnet.KindTime {
		x.ignore(reasonMalformed)
		return
	}
	x.entry.Detail, x.entry.Result = date.String()+" "+clock.String(), reqlog.ResultOK
	x.s.opts.Log(x.entry)
}

// announce sends a broadcast the device originates (I-Am, I-Have). It never
// goes to a broadcast address by itself: the asker gets it as a unicast, every
// registered foreign device as a forwarded NPDU, and the broadcast address
// only when one was configured.
func (x *exchange) announce(apdu []byte) {
	s := x.s
	global := bacnet.NPDU{HasDest: true, DNet: bacnet.GlobalBroadcast, Hops: bacnet.DefaultHopCount, APDU: apdu}.Bytes()
	foreign := s.foreigners(x.elapsed)
	var out []outgoing
	if !slices.Contains(foreign, x.from) {
		out = append(out, outgoing{x.from, x.unicast(apdu)})
	}
	s.mu.Lock()
	self := s.self
	s.mu.Unlock()
	for _, fd := range foreign {
		out = append(out, outgoing{fd, bacnet.ForwardedFrame(self, global)})
	}
	if s.opts.Broadcast.IsValid() {
		out = append(out, outgoing{s.opts.Broadcast, bacnet.BVLL{Function: bacnet.OriginalBroadcast, Data: global}})
	}
	for _, o := range out {
		x.entry.To = append(x.entry.To, o.to.String())
	}
	x.entry.Result = reqlog.ResultOK
	x.respond(out, s.device.Model().Delay(nil), nil)
}
