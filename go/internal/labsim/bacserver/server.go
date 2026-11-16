// Package bacserver is the BACnet/IP front end of lab-sim: one simulated
// device behind one UDP socket. It answers the BVLL functions of a minimal
// BBMD (foreign device registration, so a client on the same machine can get
// its broadcasts), the unconfirmed discovery services and the confirmed
// services a driver uses to learn, poll, command and subscribe to points.
// Everything a flood could grow is bounded: the packet size, the foreign
// device table, the subscriptions, the sources and the remembered answers.
package bacserver

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacobj"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// Options configure a Server. A limit left at zero gets its default.
type Options struct {
	Device *bacobj.Device
	// Elapsed is the simulator's time: behaviours, offline windows, lifetimes
	// and the rate limit all follow it.
	Elapsed func() time.Duration
	// Now stamps the log entries; nil is time.Now.
	Now func() time.Time
	// Sleep holds a delayed response; it returns false when ctx ended first.
	// nil is a real timer.
	Sleep func(ctx context.Context, d time.Duration) bool
	// Log receives every frame handled before its answer is sent; nil drops them.
	Log func(reqlog.BACnet)
	// Broadcast, when valid, also gets the broadcasts the device originates
	// (I-Am, I-Have). Without it they go to the asker and to the registered
	// foreign devices only.
	Broadcast netip.AddrPort
	// COVInterval is how often subscribed values are compared; 0 leaves it to
	// CheckCOV calls.
	COVInterval       time.Duration
	MaxForeignDevices int
	MaxSubscriptions  int
	RatePerSecond     int // packets a source address may send, sustained
	RateBurst         int // and at once
	MaxSources        int // source addresses the rate is kept for

	maxPacket int    // tests: a smaller largest frame
	handled   func() // tests: called after each datagram is dealt with
}

// Stats are a server's counters.
type Stats struct {
	Packets        int64 `json:"packets"`      // datagrams received
	RateLimited    int64 `json:"rate_limited"` // dropped: the source was over its rate
	Oversized      int64 `json:"oversized"`    // dropped: longer than a BACnet/IP frame
	Malformed      int64 `json:"malformed"`    // dropped: not a BVLL frame
	ForeignDevices int   `json:"foreign_devices"`
	Subscriptions  int   `json:"subscriptions"`
}

// Server serves one BACnet device.
type Server struct {
	opts   Options
	device *bacobj.Device

	packets, limited, oversized, malformed atomic.Int64

	mu      sync.Mutex
	conn    net.PacketConn
	self    netip.AddrPort
	limit   *limiter
	foreign *fdt
	seen    *duplicates
	subs    map[subKey]*subscription
	invoke  byte
	delayed int
	wg      sync.WaitGroup
}

func orDefault(n, fallback int) int {
	if n <= 0 {
		return fallback
	}
	return n
}

// New makes a server and tells the device which services it answers.
func New(opts Options) *Server {
	opts.MaxForeignDevices = orDefault(opts.MaxForeignDevices, defaultMaxForeignDevices)
	opts.MaxSubscriptions = orDefault(opts.MaxSubscriptions, defaultMaxSubscriptions)
	opts.RatePerSecond = orDefault(opts.RatePerSecond, defaultRate)
	opts.RateBurst = orDefault(opts.RateBurst, opts.RatePerSecond)
	opts.MaxSources = orDefault(opts.MaxSources, defaultMaxSources)
	opts.maxPacket = orDefault(opts.maxPacket, maxPacket)
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Sleep == nil {
		opts.Sleep = realSleep
	}
	if opts.Log == nil {
		opts.Log = func(reqlog.BACnet) {}
	}
	opts.Device.SetSupported(supported(opts.Device))
	return &Server{opts: opts, device: opts.Device,
		limit:   newLimiter(opts.RatePerSecond, opts.RateBurst, opts.MaxSources),
		foreign: &fdt{max: opts.MaxForeignDevices, entries: map[netip.AddrPort]registration{}},
		seen:    newDuplicates(maxDuplicates), subs: map[subKey]*subscription{}}
}

// Stats returns the counters.
func (s *Server) Stats() Stats {
	now := s.opts.Elapsed()
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{Packets: s.packets.Load(), RateLimited: s.limited.Load(), Oversized: s.oversized.Load(), Malformed: s.malformed.Load(),
		ForeignDevices: len(s.foreign.list(now)), Subscriptions: len(s.subs)}
}

// Serve answers the datagrams of conn until ctx ends, then closes conn and
// waits for the responses still held. It returns nil after such a stop, or the
// error that ended the read loop.
func (s *Server) Serve(ctx context.Context, conn net.PacketConn) error {
	s.mu.Lock()
	s.conn = conn
	if udp, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		s.self = unmap(udp.AddrPort())
	}
	s.mu.Unlock()
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-stopped:
		}
		conn.Close()
	}()
	if s.opts.COVInterval > 0 {
		s.wg.Go(func() { s.watch(ctx, stopped) })
	}
	failure := s.receive(ctx, conn)
	close(stopped)
	s.wg.Wait()
	return failure
}

// receive is the read loop.
func (s *Server) receive(ctx context.Context, conn net.PacketConn) error {
	buf := make([]byte, s.opts.maxPacket+1) // one more octet shows a datagram that is too long
	for failures := 0; ; {
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			// Windows reports an oversized datagram, and an ICMP error left by
			// an earlier send, as an error of the next read: neither ends the
			// server, but a socket that only fails does.
			if failures++; failures > maxReadErrors {
				return err
			}
			continue
		}
		failures = 0
		if udp, ok := from.(*net.UDPAddr); ok {
			s.packet(ctx, append([]byte(nil), buf[:n]...), unmap(udp.AddrPort()))
			if s.opts.handled != nil {
				s.opts.handled()
			}
		}
	}
}
