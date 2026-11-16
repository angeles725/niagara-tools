package control

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
	"github.com/angeles725/niagara-tools/go/internal/labsim/model"
)

// Listen opens the control port; an address that is not loopback is refused.
func Listen(addr string) (net.Listener, error) {
	if err := loopback.Check(addr); err != nil {
		return nil, fmt.Errorf("control address %v", err)
	}
	return net.Listen("tcp", addr)
}

// Server answers control requests about a set of devices.
type Server struct {
	Devices []*model.Device
	Elapsed func() time.Duration // the simulator's time, for a get
	Status  func() Status
	Stop    func() // asks the simulator to stop; must not block
}

// Serve answers requests on ln until ctx ends, then closes ln and waits for
// the exchanges in flight (each is bounded by a deadline).
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
		case <-stopped:
		}
		ln.Close()
	}()
	var wg sync.WaitGroup
	defer wg.Wait()
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		wg.Go(func() { s.exchange(c) })
	}
}

// exchange reads one request line and writes one response line.
func (s *Server) exchange(c net.Conn) {
	defer c.Close()
	if tcp, ok := c.RemoteAddr().(*net.TCPAddr); !ok || !tcp.IP.IsLoopback() {
		return // the listener is loopback-only; anything else is not ours to answer
	}
	if c.SetDeadline(time.Now().Add(exchangeTimeout)) != nil {
		return
	}
	line, err := bufio.NewReader(io.LimitReader(c, maxLine)).ReadBytes('\n')
	res := Response{}
	var req Request
	if err == nil {
		dec := json.NewDecoder(bytes.NewReader(line))
		dec.DisallowUnknownFields()
		err = dec.Decode(&req)
	}
	if err != nil {
		res.Error = fmt.Sprintf(errNotRequest, err)
	} else {
		res = s.answer(req)
	}
	out, _ := json.Marshal(res)
	_, _ = c.Write(append(out, '\n'))
}

func (s *Server) answer(req Request) Response {
	switch req.Op {
	case OpStatus:
		st := s.Status()
		return Response{OK: true, Status: &st}
	case OpStop:
		s.Stop()
		return Response{OK: true}
	case OpGet, OpSet:
		return s.point(req)
	}
	return Response{Error: fmt.Sprintf(errUnknownOp, req.Op)}
}

// point serves a get or a set.
func (s *Server) point(req Request) Response {
	var dev *model.Device
	var names []string
	for _, d := range s.Devices {
		names = append(names, d.Name())
		if d.Name() == req.Device {
			dev = d
		}
	}
	if dev == nil {
		return Response{Error: fmt.Sprintf(errNoDevice, req.Device, strings.Join(names, ", "))}
	}
	elapsed := s.Elapsed()
	if _, ok := dev.Get(req.Point, elapsed); !ok {
		return Response{Error: fmt.Sprintf(errNoPoint, req.Device, req.Point, strings.Join(dev.Points(), ", "))}
	}
	if req.Op == OpSet {
		if req.Value == nil {
			return Response{Error: errNoValue}
		}
		dev.Pin(req.Point, *req.Value)
	}
	v, _ := dev.Get(req.Point, elapsed)
	return Response{OK: true, Value: &v}
}
