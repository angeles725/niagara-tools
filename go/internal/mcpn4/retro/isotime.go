package retro

import (
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// hms is parse_hh_mm_ss_ff over b[p:end]: the fields and rv (negative on failure, 1 when a
// byte other than NUL follows them).
func (b isoBytes) hms(p, end int) (f [4]int, rv int) {
	sep := true
	for i := 0; i < 3; i++ {
		var ok bool
		if p, ok = b.digits(p, &f[i], 2); !ok {
			return f, -3
		}
		c := b.at(p)
		p++
		if i == 0 {
			sep = c == ':'
		}
		switch {
		case c == '.' || c == ',':
			if i < 2 || p >= end {
				return f, -3
			}
			i = 3 // break out to the fraction
		case p >= end:
			return f, boolInt(c != 0)
		case sep && c == ':':
			if i == 2 {
				return f, -4
			}
		case !sep:
			p--
		default:
			return f, -4
		}
	}
	n := min(end-p, 6)
	var ok bool
	if p, ok = b.digits(p, &f[3], n); !ok {
		return f, -3
	}
	for k := n; k < 6; k++ {
		f[3] *= 10
	}
	for isDigit(b.at(p)) {
		p++
	}
	return f, boolInt(b.at(p) != 0)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

// timeOfDay is parse_isoformat_time over b[p:]: the fields and the UTC offset in microseconds.
// Its "Z" (UTC) branch is left out: parseTS has made every Z "+00:00" before the parse.
func (b isoBytes) timeOfDay(p int) (f [4]int, offset int64, ok bool) {
	end := len(b)
	tz := p
	for {
		if c := b.at(tz); c == '+' || c == '-' {
			break
		}
		if tz++; tz >= end {
			break
		}
	}
	f, rv := b.hms(p, tz)
	switch {
	case rv < 0:
		return f, 0, false
	case tz == end:
		return f, 0, rv != 1
	}
	z, rv := b.hms(tz+1, end)
	if rv != 0 || !validTime(z[0], z[1], z[2], z[3]) {
		return f, 0, false
	}
	offset = (int64(z[0])*3600+int64(z[1])*60+int64(z[2]))*1e6 + int64(z[3])
	if b.at(tz) == '-' {
		offset = -offset
	}
	return f, offset, true
}

func validTime(h, m, s, us int) bool {
	return h >= 0 && h <= 23 && m >= 0 && m <= 59 && s >= 0 && s <= 59 && us >= 0 && us <= 999999
}

// fromISOFormat is datetime.fromisoformat(s), for an s without "Z", as microseconds since the
// epoch in UTC (a naive result taken as UTC, as _parse_ts does), and whether s parses.
func fromISOFormat(s string) (int64, bool) {
	b, ok := sanitize(s)
	if !ok {
		return 0, false
	}
	sep := b.separator()
	if sep < 0 {
		return 0, false
	}
	y, m, d, ok := b.date(sep)
	if !ok {
		return 0, false
	}
	var f [4]int
	var offset int64
	if len(b) > sep {
		p := sep + 1 // past the separator, one UTF-8 character as its lead byte sizes it
		switch c := b[sep]; {
		case c&0x80 != 0 && c&0xf0 == 0xe0:
			p = sep + 3
		case c&0x80 != 0 && c&0xf0 == 0xf0:
			p = sep + 4
		case c&0x80 != 0:
			p = sep + 2
		}
		if f, offset, ok = b.timeOfDay(p); !ok {
			return 0, false
		}
	}
	if f[0] == 24 && m >= 1 && m <= 12 && d <= daysInMonth(y, m) {
		if f[1] != 0 || f[2] != 0 || f[3] != 0 {
			return 0, false
		}
		f[0], d = 0, d+1
		if d > daysInMonth(y, m) {
			d, m = 1, m+1
			if m > 12 {
				m, y = 1, y+1
			}
		}
	}
	if y < 1 || y > 9999 || m < 1 || m > 12 || d < 1 || d > daysInMonth(y, m) || !validTime(f[0], f[1], f[2], f[3]) {
		return 0, false
	}
	t := time.Date(y, time.Month(m), d, f[0], f[1], f[2], f[3]*1000, time.UTC)
	return t.UnixMicro() - offset, true
}

// parseTS is _parse_ts: the instant of an ISO-8601 string after strip() and every "Z" made
// "+00:00"; ok is false for anything else (its ValueError).
func parseTS(v any) (int64, bool) {
	s, isStr := v.(string)
	if !isStr {
		return 0, false
	}
	return fromISOFormat(strings.ReplaceAll(strings.TrimFunc(s, pyjson.IsSpace), "Z", "+00:00"))
}
