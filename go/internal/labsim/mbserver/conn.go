package mbserver

import (
	"context"
	"net"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/modbus"
)

// serveConn answers the frames of one connection, one at a time and in order,
// until the client closes it, stays silent past the idle timeout (even in the
// middle of a frame), sends a frame whose length cannot be right, or ctx ends.
func (s *Server) serveConn(ctx context.Context, c net.Conn) {
	remote := c.RemoteAddr().String()
	for ctx.Err() == nil {
		// The deadline covers the whole next frame, so a client that sends
		// half of one cannot hold this goroutine.
		if c.SetReadDeadline(time.Now().Add(s.opts.IdleTimeout)) != nil {
			return
		}
		frame, err := modbus.ReadFrame(c)
		if err != nil {
			return
		}
		if frame.Protocol != 0 {
			continue // not Modbus: discarded, as the implementation guide asks
		}
		s.requests.Add(1)
		a := s.handle(frame, remote)
		s.opts.Log(a.entry)
		if a.delay > 0 && !s.opts.Sleep(ctx, a.delay) {
			return
		}
		if a.pdu == nil {
			continue
		}
		response := modbus.Frame{Transaction: frame.Transaction, Unit: frame.Unit, PDU: a.pdu}
		if c.SetWriteDeadline(time.Now().Add(s.opts.IdleTimeout)) != nil {
			return
		}
		if _, err := c.Write(response.Bytes()); err != nil {
			return
		}
	}
}
