package bacserver

import (
	"bytes"
	"net/netip"
	"time"
)

// The bounds of a server, and the defaults of its options.
const (
	maxPacket                = 1507 // the largest BACnet/IP frame: a 1476-octet APDU, its NPDU header and a forwarded BVLL header
	maxReadErrors            = 64   // consecutive failed reads before the server gives up
	maxDuplicates            = 64   // answers remembered for retries
	maxDelayed               = 32   // responses held at once by delay_ms
	maxLoggedValues          = 16   // values of a read written to the log
	duplicateWindow          = 10 * time.Second
	registrationGrace        = 30 * time.Second // a foreign device stays this long past its time to live (J.5.2.3)
	defaultMaxForeignDevices = 16
	defaultMaxSubscriptions  = 32
	defaultRate              = 200
	defaultMaxSources        = 64
)

// bucket is the packets a source may still send.
type bucket struct {
	tokens float64
	at     time.Duration
}

// limiter is a token bucket per source address, for a bounded number of
// sources, on the simulator's clock.
type limiter struct {
	rate, burst float64
	max         int
	sources     map[netip.Addr]*bucket
}

func newLimiter(rate, burst, sources int) *limiter {
	return &limiter{rate: float64(rate), burst: float64(burst), max: sources, sources: map[netip.Addr]*bucket{}}
}

// allow reports whether a packet from addr is handled. When the table of
// sources is full, the idle ones give their place; if none is idle the new
// source is refused, so a flood of addresses cannot grow the table.
func (l *limiter) allow(addr netip.Addr, now time.Duration) bool {
	b := l.sources[addr]
	if b == nil {
		if len(l.sources) >= l.max {
			for other, o := range l.sources {
				if o.tokens+l.rate*(now-o.at).Seconds() >= l.burst {
					delete(l.sources, other)
				}
			}
		}
		if len(l.sources) >= l.max {
			return false
		}
		b = &bucket{tokens: l.burst, at: now}
		l.sources[addr] = b
	}
	b.tokens = min(l.burst, b.tokens+l.rate*(now-b.at).Seconds())
	b.at = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// dupKey identifies a transaction: a client and its invoke id.
type dupKey struct {
	from   netip.AddrPort
	invoke byte
}

// answer is a confirmed request and the frame it was answered with.
type answer struct {
	request, payload []byte
	at               time.Duration
}

// duplicates remembers the last answers, so a request a client repeats (its
// retry after a lost or slow response) gets the same answer without running
// twice.
type duplicates struct {
	max     int
	entries map[dupKey]answer
}

func newDuplicates(max int) *duplicates {
	return &duplicates{max: max, entries: map[dupKey]answer{}}
}

// find returns the answer to the same request under the same invoke id, if it
// was given within the duplicate window.
func (d *duplicates) find(key dupKey, request []byte, now time.Duration) ([]byte, bool) {
	a, ok := d.entries[key]
	if !ok || now-a.at > duplicateWindow || !bytes.Equal(a.request, request) {
		return nil, false
	}
	return a.payload, true
}

// store remembers an answer; when the table is full the stale ones go, then
// the oldest.
func (d *duplicates) store(key dupKey, request, payload []byte, now time.Duration) {
	if _, known := d.entries[key]; !known && len(d.entries) >= d.max {
		var oldest dupKey
		first := true
		for k, a := range d.entries {
			if now-a.at > duplicateWindow {
				delete(d.entries, k)
			} else if first || a.at < d.entries[oldest].at {
				oldest, first = k, false
			}
		}
		if len(d.entries) >= d.max {
			delete(d.entries, oldest)
		}
	}
	d.entries[key] = answer{request: request, payload: payload, at: now}
}
