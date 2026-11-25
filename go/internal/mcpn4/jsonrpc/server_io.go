package jsonrpc

import (
	"bufio"
	"context"
	"errors"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Serve is Server.serve: one reply line per answered input line, flushed at once, until EOF.
// Lines end at "\n" only, as Python's stdin splits them on POSIX; a last line without one is
// still handled. A read or write error ends the loop and is returned. However the loop ends, the
// station session is closed.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	return s.ServeContext(context.Background(), in, out)
}

// ServeContext is Serve that also ends when ctx is done (Ctrl-C or SIGTERM: serve's
// KeyboardInterrupt): it returns nil at once, even while a read blocks, and writes no reply for a
// request the interrupt cut short. The station session is still closed.
func (s *Server) ServeContext(ctx context.Context, in io.Reader, out io.Writer) error {
	if s.Ctx != nil {
		defer s.Ctx.Close()
	}
	defer s.Background.Wait() // before the session closes: a running write still holds it
	type read struct {
		line []byte
		err  error
	}
	lines, done := make(chan read), make(chan struct{})
	defer close(done)
	go func() {
		r := bufio.NewReader(in)
		for {
			line, err := r.ReadBytes('\n')
			select {
			case lines <- read{line, err}:
			case <-done:
				return
			}
			if err != nil {
				return
			}
		}
	}()
	for {
		var next read
		select {
		case <-ctx.Done():
			return nil
		case next = <-lines:
		}
		if ctx.Err() != nil { // both were ready: the interrupt wins
			return nil
		}
		if len(next.line) > 0 {
			reply, ok := s.HandleLine(next.line)
			if ctx.Err() != nil {
				return nil
			}
			if ok {
				if _, werr := out.Write(append(reply, '\n')); werr != nil {
					return werr
				}
			}
			s.Background.Wait() // the next request waits for a write that answered early (#925)
		}
		if errors.Is(next.err, io.EOF) {
			return nil
		}
		if next.err != nil {
			return next.err
		}
	}
}

// HandleLine is Server.handle_line: the reply line (without "\n") and true, or false for a
// blank line or a notification.
func (s *Server) HandleLine(line []byte) ([]byte, bool) {
	line = Strip(line)
	if len(line) == 0 {
		return nil, false
	}
	var msg any
	err := errTooDeep
	if !NestsDeeper(line, MaxNesting) { // checked first: the decoder recurses per level
		msg, err = pyjson.Unmarshal(line)
	}
	var reply *pyjson.Object
	if err != nil {
		reply = errorReply(nil, ParseError, "parse error")
	} else if reply = s.Dispatch(msg); reply == nil {
		return nil, false
	}
	return encode(reply), true
}
