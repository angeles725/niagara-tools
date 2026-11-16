package bacserver

import (
	"context"
	"net/netip"
	"time"
)

// watch compares the subscribed values on a timer.
func (s *Server) watch(ctx context.Context, stopped <-chan struct{}) {
	ticker := time.NewTicker(s.opts.COVInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopped:
			return
		case <-ticker.C:
			s.CheckCOV()
		}
	}
}

func unmap(a netip.AddrPort) netip.AddrPort { return netip.AddrPortFrom(a.Addr().Unmap(), a.Port()) }

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
