package bacserver

import (
	"encoding/binary"
	"fmt"
	"net/netip"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// linkFunctions says what each BVLL function a device or a BBMD receives does.
var linkFunctions map[byte]func(x *exchange, data []byte)

// ignoredLinks are the BVLL functions that exist but get no answer here: the
// answers themselves, and what only a peer BBMD would send (there is none).
var ignoredLinks = map[byte]struct{}{
	bacnet.BVLCResult: {}, bacnet.ReadBDTAck: {}, bacnet.ReadFDTAck: {}, bacnet.ForwardedNPDU: {},
}

func init() {
	linkFunctions = map[byte]func(x *exchange, data []byte){
		bacnet.OriginalUnicast:       (*exchange).network,
		bacnet.OriginalBroadcast:     (*exchange).network,
		bacnet.DistributeBroadcast:   (*exchange).distribute,
		bacnet.RegisterForeignDevice: (*exchange).register,
		bacnet.ReadFDT:               (*exchange).readFDT,
		bacnet.DeleteFDTEntry:        (*exchange).deleteFDT,
		bacnet.ReadBDT:               func(x *exchange, _ []byte) { x.link(bacnet.BVLL{Function: bacnet.ReadBDTAck}, "") },
		bacnet.WriteBDT:              func(x *exchange, _ []byte) { x.result(bacnet.ResultWriteBDTNAK, reasonNotSupported) },
	}
}

// link answers a BVLL function with one frame, unless the device is offline. A
// reason marks a refusal.
func (x *exchange) link(frame bacnet.BVLL, reason string) {
	if x.offline() {
		return
	}
	x.entry.Result = reqlog.ResultOK
	if reason != "" {
		x.entry.Result, x.entry.Reason = reqlog.ResultRefused, reason
	}
	x.respond([]outgoing{{x.from, frame}}, 0, nil)
}

// result answers with a BVLC-Result.
func (x *exchange) result(code uint16, reason string) { x.link(bacnet.ResultFrame(code), reason) }

// foreigners lists the registered foreign devices now.
func (s *Server) foreigners(now time.Duration) []netip.AddrPort {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []netip.AddrPort
	for _, e := range s.foreign.list(now) {
		out = append(out, e.Address)
	}
	return out
}

// register serves Register-Foreign-Device.
func (x *exchange) register(data []byte) {
	if len(data) != 2 {
		x.result(bacnet.ResultRegisterNAK, reasonMalformed)
		return
	}
	ttl := binary.BigEndian.Uint16(data)
	x.entry.Detail = fmt.Sprintf("ttl %d s", ttl)
	if x.offline() {
		return
	}
	x.s.mu.Lock()
	ok := x.from.Addr().Is4() && x.s.foreign.register(x.from, ttl, x.elapsed)
	x.s.mu.Unlock()
	if !ok {
		x.result(bacnet.ResultRegisterNAK, reasonTableFull)
		return
	}
	x.result(bacnet.ResultSuccess, "")
}

// readFDT serves Read-Foreign-Device-Table.
func (x *exchange) readFDT([]byte) {
	x.s.mu.Lock()
	entries := x.s.foreign.list(x.elapsed)
	x.s.mu.Unlock()
	x.link(bacnet.ReadFDTAckFrame(entries), "")
}

// deleteFDT serves Delete-Foreign-Device-Table-Entry.
func (x *exchange) deleteFDT(data []byte) {
	addr, rest, ok := bacnet.ParseForwarded(data)
	if !ok || len(rest) != 0 {
		x.result(bacnet.ResultDeleteFDTNAK, reasonMalformed)
		return
	}
	if x.offline() {
		return
	}
	x.s.mu.Lock()
	removed := x.s.foreign.remove(addr, x.elapsed)
	x.s.mu.Unlock()
	if !removed {
		x.result(bacnet.ResultDeleteFDTNAK, reasonNoEntry)
		return
	}
	x.result(bacnet.ResultSuccess, "")
}

// distribute serves Distribute-Broadcast-To-Network: the NPDU of a registered
// foreign device goes on to the other foreign devices and to the broadcast
// address, then to the simulated device itself.
func (x *exchange) distribute(data []byte) {
	if x.offline() {
		return
	}
	others := x.s.foreigners(x.elapsed)
	registered := false
	for _, fd := range others {
		registered = registered || fd == x.from
	}
	if !registered {
		x.result(bacnet.ResultDistributeNAK, reasonNotForeign)
		return
	}
	forwarded := bacnet.ForwardedFrame(x.from, data)
	for _, fd := range others {
		if fd != x.from {
			x.s.send(fd, forwarded)
		}
	}
	x.s.send(x.s.opts.Broadcast, forwarded)
	x.network(data)
}
