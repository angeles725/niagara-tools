package bacserver

import (
	"net/netip"
	"slices"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/labsim/bacnet"
)

// registration is one foreign device.
type registration struct {
	ttl     uint16
	expires time.Duration
}

// fdt is the foreign device table of the simulator's minimal BBMD.
type fdt struct {
	max     int
	entries map[netip.AddrPort]registration
}

func (f *fdt) purge(now time.Duration) {
	for addr, r := range f.entries {
		if now >= r.expires {
			delete(f.entries, addr)
		}
	}
}

// register adds or renews a foreign device for ttl seconds and the grace
// period; it fails when the table is full of other devices.
func (f *fdt) register(addr netip.AddrPort, ttl uint16, now time.Duration) bool {
	f.purge(now)
	if _, known := f.entries[addr]; !known && len(f.entries) >= f.max {
		return false
	}
	f.entries[addr] = registration{ttl: ttl, expires: now + time.Duration(ttl)*time.Second + registrationGrace}
	return true
}

// remove deletes an entry and reports whether it was there.
func (f *fdt) remove(addr netip.AddrPort, now time.Duration) bool {
	f.purge(now)
	_, known := f.entries[addr]
	delete(f.entries, addr)
	return known
}

// list is the table now, in address order.
func (f *fdt) list(now time.Duration) []bacnet.FDTEntry {
	f.purge(now)
	out := make([]bacnet.FDTEntry, 0, len(f.entries))
	for addr, r := range f.entries {
		left := (r.expires - now + time.Second - 1) / time.Second
		out = append(out, bacnet.FDTEntry{Address: addr, TTL: r.ttl, Remaining: uint16(min(left, 0xFFFF))})
	}
	slices.SortFunc(out, func(a, b bacnet.FDTEntry) int { return a.Address.Compare(b.Address) })
	return out
}
