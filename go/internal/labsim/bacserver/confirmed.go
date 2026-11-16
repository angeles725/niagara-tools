package bacserver

import (
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// call is a decoded confirmed request, ready to run.
type call struct {
	points []string // the model points it touches: their delay and their fail_after_reads apply
	read   bool     // it counts as a read against fail_after_reads
	// run executes it: the service data of a ComplexACK, nil for a SimpleACK,
	// or the error the device answers.
	run func(x *exchange) ([]byte, *bacnet.Error)
	// after, when set, runs once the acknowledgement of a successful call is sent.
	after func(s *Server)
}

// service is one row of a service table.
type service struct {
	supported []int                                 // its bits of protocol-services-supported
	enabled   func(d *bacobj.Device) bool           // nil: always
	prepare   func(*exchange, []byte) (call, error) // confirmed: decodes the request (the error is a bacnet.Reject)
	handle    func(*exchange, []byte)               // unconfirmed
}

func covEnabled(d *bacobj.Device) bool { return d.Spec().COV }

// confirmedServices are the confirmed services the simulator executes; any
// other is rejected as unrecognized.
var confirmedServices = map[byte]service{
	bacnet.ServiceReadProperty:         {supported: []int{bacnet.SupportedReadProperty}, prepare: prepareRead},
	bacnet.ServiceReadPropertyMultiple: {supported: []int{bacnet.SupportedReadPropertyMultiple}, prepare: prepareReadMultiple},
	bacnet.ServiceWriteProperty:        {supported: []int{bacnet.SupportedWriteProperty}, prepare: prepareWrite},
	bacnet.ServiceSubscribeCOV: {supported: []int{bacnet.SupportedSubscribeCOV, bacnet.SupportedConfirmedCOVNotification,
		bacnet.SupportedUnconfirmedCOVNotification}, enabled: covEnabled, prepare: prepareSubscribe},
}

// supported lists the bits of protocol-services-supported for a device: what
// the two service tables implement and the device enables.
func supported(d *bacobj.Device) []int {
	var bits []int
	for _, table := range []map[byte]service{confirmedServices, unconfirmedServices} {
		for _, svc := range table {
			if svc.enabled == nil || svc.enabled(d) {
				bits = append(bits, svc.supported...)
			}
		}
	}
	return bits
}

// outcome writes what an answer APDU says into the log entry.
func (x *exchange) outcome(apdu []byte) {
	a, _ := bacnet.ParseAPDU(apdu)
	switch a.Type {
	case bacnet.PDUError:
		x.entry.Result, x.entry.ErrorClass, x.entry.ErrorCode = reqlog.ResultError, a.Err.ClassName(), a.Err.CodeName()
	case bacnet.PDUReject:
		x.entry.Result, x.entry.Reason = reqlog.ResultReject, bacnet.Reject(a.Reason).String()
	case bacnet.PDUAbort:
		x.entry.Result, x.entry.Reason = reqlog.ResultAbort, bacnet.Abort(a.Reason).String()
	default:
		x.entry.Result = reqlog.ResultOK
	}
}

// replay answers a repeated request with the frame its first copy got.
func (x *exchange) replay(key dupKey) bool {
	x.s.mu.Lock()
	payload, found := x.s.seen.find(key, x.npdu.APDU, x.elapsed)
	x.s.mu.Unlock()
	if !found {
		return false
	}
	frame, _ := bacnet.ParseBVLL(payload)
	if n, ok := bacnet.ParseNPDU(frame.Data); ok {
		x.outcome(n.APDU)
	}
	x.entry.Duplicate = true
	x.respond([]outgoing{{x.from, frame}}, 0, nil)
	return true
}

// confirmed answers a confirmed request, in the order a device decides: is it
// online, is this a retry, is the service known and the request well formed,
// has the device or a touched point failed; only then is it executed. The
// answer must fit what both ends accept, for there is no segmentation.
func (x *exchange) confirmed(a bacnet.APDU) {
	s, device := x.s, x.s.device
	invoke := int(a.InvokeID)
	x.entry.InvokeID, x.entry.Service = &invoke, bacnet.ServiceName(true, a.Service)
	svc, known := confirmedServices[a.Service]
	var c call
	var malformed error
	if known && !a.Segmented {
		c, malformed = svc.prepare(x, a.Data)
	}
	key := dupKey{from: x.from, invoke: a.InvokeID}
	if x.offline() || x.replay(key) {
		return
	}
	delay := device.Model().Delay(c.points)
	var apdu []byte
	var after func()
	switch reject, _ := malformed.(bacnet.Reject); {
	case a.Segmented:
		apdu = bacnet.AbortPDU(a.InvokeID, bacnet.AbortSegmentationNotSupported)
	case !known:
		apdu = bacnet.RejectPDU(a.InvokeID, bacnet.RejectUnrecognizedService)
	case malformed != nil:
		apdu = bacnet.RejectPDU(a.InvokeID, reject)
	case !device.Model().Admit(c.points, c.read):
		apdu = bacnet.ErrorPDU(a.InvokeID, a.Service, *bacnet.ErrOperationalProblem)
	default:
		data, err := c.run(x)
		switch {
		case err != nil:
			apdu = bacnet.ErrorPDU(a.InvokeID, a.Service, *err)
		case data == nil:
			apdu = bacnet.SimpleACK(a.InvokeID, a.Service)
		default:
			apdu = bacnet.ComplexACK(a.InvokeID, a.Service, data)
		}
		if err == nil && c.after != nil {
			after = func() { c.after(s) }
		}
	}
	if len(apdu) > min(a.MaxAPDU, device.Spec().MaxAPDU) {
		apdu, after = bacnet.AbortPDU(a.InvokeID, bacnet.AbortSegmentationNotSupported), nil
		x.entry.Read = nil
	}
	x.outcome(apdu)
	frame := x.unicast(apdu)
	s.mu.Lock()
	s.seen.store(key, x.npdu.APDU, frame.Bytes(), x.elapsed)
	s.mu.Unlock()
	x.respond([]outgoing{{x.from, frame}}, delay, after)
}
