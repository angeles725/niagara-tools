package scenario

import (
	"fmt"
	"net"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/labsim/loopback"
)

// faults collects messages, each prefixed with the place it concerns.
type faults struct {
	list []string
}

func (f *faults) add(where, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if where != "" {
		msg = where + ": " + msg
	}
	f.list = append(f.list, msg)
}

// label names the n-th (0-based) item of a kind: `device 2 "unit-b"`.
func label(kind string, n int, name string) string {
	if name == "" {
		return fmt.Sprintf("%s %d", kind, n+1)
	}
	return fmt.Sprintf("%s %d %q", kind, n+1, name)
}

// Validate returns one message per fault of the scenario, in file order; a
// scenario with none can be built and served.
func Validate(f *File) []string {
	out := &faults{}
	validateTop(f, out)
	names, units := map[string]bool{}, map[int]string{}
	seen := &bacnetSeen{instances: map[int]string{}, listens: map[string]string{}}
	for i := range f.Devices {
		d := &f.Devices[i]
		where := label("device", i, d.Name)
		switch {
		case d.Name == "":
			out.add(where, msgNoName)
		case names[d.Name]:
			out.add(where, msgNameUsed, d.Name)
		}
		names[d.Name] = true
		if d.Unit < MinUnit || d.Unit > MaxUnit {
			out.add(where, msgUnitRange, d.Unit, MinUnit, MaxUnit)
		} else if owner, used := units[d.Unit]; used {
			out.add(where, msgUnitUsed, d.Unit, owner)
		} else {
			units[d.Unit] = d.Name
		}
		validateDeviceFaults(d.Faults, where, out)
		if len(d.Points) == 0 {
			out.add(where, msgNoPoints)
		}
		validatePoints(d, f.dir, where, out)
		validateBACnet(d, where, seen, out)
	}
	return out.list
}

func validateTop(f *File, out *faults) {
	if len(f.Devices) == 0 {
		out.add("", msgNoDevices)
	}
	if !hostPort(f.Listen) {
		out.add("", msgListen, f.Listen)
	}
	if err := loopback.Check(f.Control); err != nil {
		out.add("", msgControl, err)
	}
	if f.UnknownUnit != UnknownSilent && f.UnknownUnit != UnknownGateway {
		out.add("", msgUnknownUnit, f.UnknownUnit)
	}
	if f.MaxConnections < 1 || f.MaxConnections > LimitConnections {
		out.add("", msgMaxConnections, f.MaxConnections, LimitConnections)
	}
	if f.IdleTimeoutMS < 0 {
		out.add("", msgIdleTimeout, f.IdleTimeoutMS)
	}
}

// hostPort reports whether addr is host:port with a numeric port.
func hostPort(addr string) bool {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	n, err := strconv.Atoi(port)
	return err == nil && n >= 0 && n <= 65535
}

func validateDeviceFaults(f *Faults, where string, out *faults) {
	if f == nil {
		return
	}
	validateCounters(f.FailAfterReads, f.DelayMS, where, out)
	for _, w := range f.Offline {
		if w.FromMS < 0 || w.ToMS <= w.FromMS {
			out.add(where, msgOfflineWindow, w.FromMS, w.ToMS)
		}
	}
}

// validateCounters checks the two faults a device and a point share.
func validateCounters(failAfter *int, delayMS int, where string, out *faults) {
	if failAfter != nil && *failAfter < 0 {
		out.add(where, msgFailAfter, *failAfter)
	}
	if delayMS < 0 {
		out.add(where, msgDelay, delayMS)
	}
}
