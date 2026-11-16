package bacserver

import (
	"context"
	"net"
	"net/netip"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// Why a frame is ignored, as the log's reason.
const (
	reasonUnknownLink  = "unknown_bvll_function"
	reasonNotAnswered  = "not_answered" // a BVLL function a device only receives as an answer, or from a peer BBMD
	reasonBadNPDU      = "malformed_npdu"
	reasonBadAPDU      = "malformed_apdu"
	reasonNetworkLayer = "network_layer_message"
	reasonOtherNetwork = "other_network"
	reasonNotRequest   = "not_a_request"
	reasonUnsupported  = "unsupported_service"
	reasonNotInRange   = "not_in_range"
	reasonNoObject     = "no_such_object"
	reasonBusy         = "too_many_held_responses"
	reasonNotForeign   = "not_a_foreign_device"
	reasonTableFull    = "table_full"
	reasonMalformed    = "malformed"
	reasonNotSupported = "not_supported"
	reasonNoEntry      = "no_such_entry"
)

// exchange is one received frame on its way to an answer.
type exchange struct {
	s       *Server
	ctx     context.Context
	from    netip.AddrPort
	elapsed time.Duration
	npdu    bacnet.NPDU
	entry   reqlog.BACnet
}

// outgoing is one frame to send.
type outgoing struct {
	to    netip.AddrPort
	frame bacnet.BVLL
}

// newEntry starts the log entry of a frame from or to remote.
func (s *Server) newEntry(remote netip.AddrPort, link byte, elapsed time.Duration) reqlog.BACnet {
	return reqlog.BACnet{Time: s.opts.Now().UTC().Format(time.RFC3339Nano), ElapsedMS: elapsed.Milliseconds(),
		Protocol: reqlog.ProtocolBACnetIP, Remote: remote.String(), Device: s.device.Spec().Name, Link: bacnet.BVLLName(link)}
}

// packet handles one datagram.
func (s *Server) packet(ctx context.Context, p []byte, from netip.AddrPort) {
	s.packets.Add(1)
	now := s.opts.Elapsed()
	s.mu.Lock()
	allowed := s.limit.allow(from.Addr(), now)
	s.mu.Unlock()
	if !allowed {
		s.limited.Add(1)
		return
	}
	if len(p) > s.opts.maxPacket {
		s.oversized.Add(1)
		return
	}
	frame, ok := bacnet.ParseBVLL(p)
	if !ok {
		s.malformed.Add(1)
		return
	}
	x := &exchange{s: s, ctx: ctx, from: from, elapsed: now, entry: s.newEntry(from, frame.Function, now)}
	if handle, known := linkFunctions[frame.Function]; known {
		handle(x, frame.Data)
	} else if _, named := ignoredLinks[frame.Function]; named {
		x.ignore(reasonNotAnswered)
	} else {
		x.ignore(reasonUnknownLink)
	}
}

// offline logs the frame as unanswered when the device is in an offline
// window, and reports it.
func (x *exchange) offline() bool {
	if !x.s.device.Model().Offline(x.elapsed) {
		return false
	}
	x.entry.Result, x.entry.Reason = reqlog.ResultNoResponse, reqlog.ReasonOffline
	x.s.opts.Log(x.entry)
	return true
}

// ignore logs a frame a device does not answer.
func (x *exchange) ignore(reason string) {
	x.entry.Result, x.entry.Reason = reqlog.ResultIgnored, reason
	x.s.opts.Log(x.entry)
}

// respond logs the exchange and sends its frames, at once or after delay; then
// runs after. Held responses are bounded: one too many is not sent.
func (x *exchange) respond(out []outgoing, delay time.Duration, after func()) {
	s := x.s
	x.entry.DelayMS = delay.Milliseconds()
	deliver := func() {
		for _, o := range out {
			s.send(o.to, o.frame)
		}
		if after != nil {
			after()
		}
	}
	if delay <= 0 {
		s.opts.Log(x.entry)
		deliver()
		return
	}
	s.mu.Lock()
	room := s.delayed < maxDelayed
	if room {
		s.delayed++
	}
	s.mu.Unlock()
	if !room {
		x.entry.Result, x.entry.Reason = reqlog.ResultNoResponse, reasonBusy
		s.opts.Log(x.entry)
		return
	}
	s.opts.Log(x.entry)
	s.wg.Go(func() {
		if s.opts.Sleep(x.ctx, delay) {
			deliver()
		}
		s.mu.Lock()
		s.delayed--
		s.mu.Unlock()
	})
}

// send writes one frame; a failed send is dropped, as a lost datagram is.
func (s *Server) send(to netip.AddrPort, frame bacnet.BVLL) {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil || !to.IsValid() {
		return
	}
	_, _ = conn.WriteTo(frame.Bytes(), net.UDPAddrFromAddrPort(to))
}

// unicast wraps an APDU for the client of this exchange.
func (x *exchange) unicast(apdu []byte) bacnet.BVLL {
	return bacnet.BVLL{Function: bacnet.OriginalUnicast, Data: x.npdu.Reply(apdu).Bytes()}
}
