package labsim

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// probeInvoke is the invoke id of every request of the probe: it makes one
// confirmed request per run.
const probeInvoke = 1

// probeAction is one action of bacnet-probe: how many arguments it takes, the
// usage text for a wrong count, and what it does. prepare returns the function
// that talks to the simulator, or the usage error of its arguments.
type probeAction struct {
	min, max int
	args     string
	prepare  func(p parsed, args []string) (func(c *probeClient) (int, error), error)
}

var probeActions map[string]probeAction

func init() {
	probeActions = map[string]probeAction{
		"who-is":        {0, 2, argsWhoIs, prepareWhoIs},
		"read":          {2, 3, argsRead, prepareProbeRead},
		"read-multiple": {2, 2, argsReadMultiple, prepareProbeReadMultiple},
		"write":         {2, 2, argsWrite, prepareProbeWrite},
		"subscribe":     {1, 1, argsSubscribe, prepareProbeSubscribe},
	}
}

// bacnetProbeCommand sends BACnet requests to a simulator on this machine and
// prints each answer as one JSON line. Like probe, it is a way to see the
// simulator answer, not a field tool: an address that is not loopback is
// refused.
func bacnetProbeCommand(p parsed, s cli.Stdio) (int, error) {
	action, known := probeActions[p.rest[0]]
	if !known {
		return exitUsage, fmt.Errorf(msgProbeAction, p.rest[0])
	}
	args := p.rest[1:]
	if len(args) < action.min || len(args) > action.max {
		return exitUsage, fmt.Errorf(msgProbeArgs, p.rest[0], action.args)
	}
	timeout, err := whole(p, "timeout-ms", "3000", 1, 600000)
	if err != nil {
		return exitUsage, err
	}
	maxAPDU, err := whole(p, "max-apdu", "1476", 0, bacnet.MaxAPDU)
	if err != nil || !bacnet.ValidMaxAPDU(maxAPDU) {
		return exitUsage, fmt.Errorf(msgProbeMaxAPDU, p.option("max-apdu", ""))
	}
	run, err := action.prepare(p, args)
	if err != nil {
		return exitUsage, err
	}
	addr := p.option("addr", scenario.DefaultBACnetListen)
	if err := loopback.Check(addr); err != nil {
		return exitFailed, fmt.Errorf("probe address %v", err)
	}
	server, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return exitFailed, err
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		return exitFailed, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Millisecond)); err != nil {
		return exitFailed, err
	}
	target := server.AddrPort()
	client := &probeClient{conn: conn, server: netip.AddrPortFrom(target.Addr().Unmap(), target.Port()), maxAPDU: maxAPDU, out: s.Out}
	code, err := run(client)
	var timedOut net.Error
	if errors.As(err, &timedOut) && timedOut.Timeout() {
		return exitFailed, fmt.Errorf(msgProbeNoAnswer, addr, timeout, err)
	}
	return code, err
}

// probeClient is the probe's end of the exchange: one UDP socket on loopback
// with one deadline for the whole run.
type probeClient struct {
	conn    *net.UDPConn
	server  netip.AddrPort
	maxAPDU int
	out     io.Writer
}

// send sends an APDU to the simulator as an Original-Unicast-NPDU.
func (c *probeClient) send(apdu []byte, expectingReply bool) error {
	frame := bacnet.BVLL{Function: bacnet.OriginalUnicast, Data: bacnet.NPDU{ExpectingReply: expectingReply, APDU: apdu}.Bytes()}
	_, err := c.conn.WriteToUDPAddrPort(frame.Bytes(), c.server)
	return err
}

// next is the next APDU the simulator sent; frames from anywhere else, and
// frames that carry no APDU, are passed over.
func (c *probeClient) next() (bacnet.APDU, error) {
	buf := make([]byte, 2048)
	for {
		n, from, err := c.conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			return bacnet.APDU{}, err
		}
		frame, ok := bacnet.ParseBVLL(buf[:n])
		if !ok || netip.AddrPortFrom(from.Addr().Unmap(), from.Port()) != c.server {
			continue
		}
		data := frame.Data
		if frame.Function == bacnet.ForwardedNPDU {
			_, data, _ = bacnet.ParseForwarded(data)
		}
		if npdu, ok := bacnet.ParseNPDU(data); ok && !npdu.Network {
			if apdu, ok := bacnet.ParseAPDU(npdu.APDU); ok {
				return apdu, nil
			}
		}
	}
}

// confirmed sends a confirmed request and returns its answer.
func (c *probeClient) confirmed(service byte, data []byte) (bacnet.APDU, error) {
	if err := c.send(bacnet.ConfirmedRequest(c.maxAPDU, probeInvoke, service, data), true); err != nil {
		return bacnet.APDU{}, err
	}
	for {
		a, err := c.next()
		if err != nil {
			return a, err
		}
		if a.Type != bacnet.PDUConfirmed && a.Type != bacnet.PDUUnconfirmed && a.InvokeID == probeInvoke {
			return a, nil
		}
	}
}

// print writes one JSON line.
func (c *probeClient) print(line probeLine) {
	out, _ := json.Marshal(line)
	fmt.Fprintf(c.out, "%s\n", out)
}

// finish prints the line of a confirmed request: with what a refusal says
// (exit 1), or as it is.
func (c *probeClient) finish(line probeLine, a bacnet.APDU) (int, error) {
	switch a.Type {
	case bacnet.PDUError:
		line.Error = &probeError{Class: a.Err.ClassName(), Code: a.Err.CodeName()}
	case bacnet.PDUReject:
		line.Reject = bacnet.Reject(a.Reason).String()
	case bacnet.PDUAbort:
		line.Abort = bacnet.Abort(a.Reason).String()
	default:
		c.print(line)
		return exitOK, nil
	}
	line.Values, line.Results, line.Result, line.Subscribed, line.Lifetime, line.Process = nil, nil, "", "", nil, nil
	c.print(line)
	return exitFailed, nil
}
