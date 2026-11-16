// Package mbserver is the Modbus TCP server of lab-sim: it accepts a bounded
// number of connections, reads frames with a deadline, and answers each request
// from the register map of the unit it addresses.
package mbserver

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/regmap"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
)

// The values an Options field gets when it is left at zero.
const (
	defaultMaxConnections = 8
	defaultIdleTimeout    = time.Minute
)

// Options configure a Server.
type Options struct {
	Units map[byte]*regmap.Unit // the simulated slaves by unit id
	// GatewayException answers a unit id no device has with exception 0x0B;
	// otherwise such a request gets no response.
	GatewayException bool
	MaxConnections   int           // connections served at once; one more is closed at once
	IdleTimeout      time.Duration // how long a connection may take to deliver its next whole frame
	// Clock is the time behaviours and offline windows follow; nil is time.Now.
	Clock func() time.Time
	// Sleep holds a delayed response; it returns false when ctx ended first.
	// nil is a real timer.
	Sleep func(ctx context.Context, d time.Duration) bool
	// Log receives every request before its response is sent; nil drops them.
	Log func(reqlog.Entry)
}

// Stats are a server's counters.
type Stats struct {
	Requests    int64 `json:"requests"`    // frames handled, answered or not
	Connections int   `json:"connections"` // open now
	Refused     int64 `json:"refused"`     // connections closed for being over the limit
}

// Server is a Modbus TCP server over a set of simulated units.
type Server struct {
	opts  Options
	start time.Time

	requests atomic.Int64
	refused  atomic.Int64
	mu       sync.Mutex
	conns    map[net.Conn]struct{}
	wg       sync.WaitGroup
}

// New makes a server; its clock starts now.
func New(opts Options) *Server {
	if opts.MaxConnections <= 0 {
		opts.MaxConnections = defaultMaxConnections
	}
	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = defaultIdleTimeout
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	if opts.Sleep == nil {
		opts.Sleep = realSleep
	}
	if opts.Log == nil {
		opts.Log = func(reqlog.Entry) {}
	}
	return &Server{opts: opts, start: opts.Clock(), conns: map[net.Conn]struct{}{}}
}

// Elapsed is the time since the server was made, on its clock.
func (s *Server) Elapsed() time.Duration { return s.opts.Clock().Sub(s.start) }

// Stats returns the counters.
func (s *Server) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Stats{Requests: s.requests.Load(), Connections: len(s.conns), Refused: s.refused.Load()}
}

// Serve accepts connections on ln until ctx ends, then closes the listener and
// every open connection and waits for their goroutines. It returns nil after
// such a stop, or the error that ended the accept loop.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
		case <-stopped:
		}
		ln.Close()
		s.closeAll()
	}()
	var failure error
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() == nil {
				failure = err
			}
			break
		}
		if !s.track(c) {
			s.refused.Add(1)
			c.Close()
			continue
		}
		s.wg.Go(func() {
			defer s.untrack(c)
			s.serveConn(ctx, c)
		})
	}
	ln.Close()
	s.closeAll()
	s.wg.Wait()
	return failure
}

// track registers a connection unless the server is already at its limit.
func (s *Server) track(c net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.conns) >= s.opts.MaxConnections {
		return false
	}
	s.conns[c] = struct{}{}
	return true
}

func (s *Server) untrack(c net.Conn) {
	c.Close()
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, c)
}

// closeAll closes every open connection, which ends its goroutine's read.
func (s *Server) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.conns {
		c.Close()
	}
}

// realSleep waits d on a timer, or until ctx ends.
func realSleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-ctx.Done():
		return false
	}
}
