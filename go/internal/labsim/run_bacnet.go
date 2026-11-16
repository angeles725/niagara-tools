package labsim

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacserver"
	"github.com/angeles725/niagara-tools/go/internal/labsim/control"
	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
	"github.com/angeles725/niagara-tools/go/internal/labsim/reqlog"
	"github.com/angeles725/niagara-tools/go/internal/labsim/scenario"
)

// covInterval is how often a running simulator compares the subscribed values
// that change by themselves (behaviours).
const covInterval = 500 * time.Millisecond

// ipv4 reads an IPv4 address and a port.
func ipv4(text string) (netip.AddrPort, bool) {
	addr, err := netip.ParseAddrPort(text)
	return addr, err == nil && addr.Addr().Is4()
}

// broadcastAddress reads --bacnet-broadcast: where the broadcasts the devices
// originate also go. Without it they never leave for a broadcast address.
func broadcastAddress(p parsed) (netip.AddrPort, error) {
	text, given := p.options["bacnet-broadcast"]
	if !given {
		return netip.AddrPort{}, nil
	}
	addr, ok := ipv4(text)
	if !ok {
		return addr, fmt.Errorf(msgBadBroadcast, text)
	}
	return addr, nil
}

// bacnetAddresses decides where each BACnet device listens: the scenario's
// address when it is loopback, or what --bacnet-listen names for the device
// ("device=host:port", several separated by commas; a bare "host:port" when
// the scenario has one BACnet device). As for Modbus, a scenario alone never
// binds outside this machine.
func bacnetAddresses(f *scenario.File, flag string, given bool) (map[string]string, error) {
	scenarioListen, flagged := map[string]string{}, map[string]string{}
	only := ""
	for i := range f.Devices {
		if d := &f.Devices[i]; d.BACnet != nil {
			scenarioListen[d.Name], only = d.BACnet.Listen, d.Name
		}
	}
	for _, item := range strings.Split(flag, ",") {
		if !given {
			break
		}
		name, addr, named := strings.Cut(item, "=")
		if !named {
			if len(scenarioListen) != 1 {
				return nil, fmt.Errorf(msgBACnetWhich, item, len(scenarioListen))
			}
			name, addr = only, item
		}
		if _, known := scenarioListen[name]; !known {
			return nil, fmt.Errorf(msgBACnetNoDevice, name)
		}
		if _, twice := flagged[name]; twice {
			return nil, fmt.Errorf(msgBACnetTwice, name)
		}
		if _, ok := ipv4(addr); !ok {
			return nil, fmt.Errorf(msgBadBACnetListen, addr)
		}
		flagged[name] = addr
	}
	out := map[string]string{}
	for name, addr := range scenarioListen {
		if chosen, ok := flagged[name]; ok {
			out[name] = chosen
			continue
		}
		if err := loopback.Check(addr); err != nil {
			return nil, fmt.Errorf(msgBACnetExposed, name, err, name, addr)
		}
		out[name] = addr
	}
	return out, nil
}

// fieldEnds are the BACnet front ends of a run.
type fieldEnds struct {
	ends    []frontEnd
	changed func() // tells every device that a point was set behind the protocol
}

// bacnetEnds opens one UDP socket per BACnet device and makes its server. The
// ends opened before a failure are returned with it, to be closed.
func bacnetEnds(built scenario.Built, listen map[string]string, broadcast netip.AddrPort, elapsed func() time.Duration, logTo *reqlog.Writer) (fieldEnds, error) {
	var out fieldEnds
	var servers []*bacserver.Server
	out.changed = func() {
		for _, server := range servers {
			server.CheckCOV()
		}
	}
	for _, end := range built.BACnet {
		addr, _ := ipv4(listen[end.Name])
		// "udp4": BACnet/IP is IPv4. No address reuse is asked, so a port a
		// station already holds on the same address is refused, not shared.
		conn, err := net.ListenUDP("udp4", net.UDPAddrFromAddrPort(addr))
		if err != nil {
			return out, fmt.Errorf(msgBACnetBind, end.Name, err)
		}
		server := bacserver.New(bacserver.Options{Device: end.Device, Elapsed: elapsed, Log: logTo.LogBACnet,
			Broadcast: broadcast, COVInterval: covInterval})
		servers = append(servers, server)
		out.ends = append(out.ends, frontEnd{
			FrontEnd: control.FrontEnd{Protocol: protocolBACnet, Address: conn.LocalAddr().String(), Device: end.Name},
			serve:    func(ctx context.Context) error { return server.Serve(ctx, conn) },
			close:    func() { conn.Close() },
		})
	}
	return out, nil
}
