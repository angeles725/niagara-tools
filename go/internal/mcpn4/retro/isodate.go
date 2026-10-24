package retro

import (
	"time"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// This file is datetime.fromisoformat as CPython 3.15 runs it (Modules/_datetimemodule.c:
// datetime_datetime_fromisoformat_impl and its parse_* helpers): the C code walks the UTF-8
// bytes and reads the terminating NUL past their end, so at() answers 0 there and an embedded
// NUL stops a parse the same way. Only the instant matters to the retro: every failure is the
// same ValueError to _parse_ts.

// isoBytes is the parse buffer: the UTF-8 of the string, NUL past its end.
type isoBytes []byte

func (b isoBytes) at(i int) byte {
	if i < len(b) {
		return b[i]
	}
	return 0
}

func isDigit(c byte) bool { return c-'0' < 10 }

// digits is parse_digits: n ASCII digits at i into *v, and the index after them; ok is false at
// the first other byte (the digits before it stay in *v, as in C).
func (b isoBytes) digits(i int, v *int, n int) (int, bool) {
	for k := 0; k < n; k++ {
		c := b.at(i + k)
		if !isDigit(c) {
			return 0, false
		}
		*v = *v*10 + int(c-'0')
	}
	return i + n, true
}

// sanitize is _sanitize_isoformat_str plus PyUnicode_AsUTF8AndSize: fewer than 7 code points
// or a lone surrogate anywhere but the first separator position (7, 8 or 10) holding one fails;
// that one becomes 'T'.
func sanitize(s string) (isoBytes, bool) {
	var cps []rune
	for i := 0; i < len(s); {
		r, n := pyjson.DecodeRune(s[i:])
		cps = append(cps, r)
		i += n
	}
	if len(cps) < 7 {
		return nil, false
	}
	for _, pos := range []int{7, 8, 10} {
		if pos > len(cps) {
			break
		}
		if pos < len(cps) && isSurrogate(cps[pos]) {
			cps[pos] = 'T'
			break
		}
	}
	out := make([]byte, 0, len(s))
	for _, r := range cps {
		if isSurrogate(r) {
			return nil, false
		}
		out = utf8.AppendRune(out, r)
	}
	return out, true
}

func isSurrogate(r rune) bool { return r >= 0xd800 && r <= 0xdfff }

// separator is _find_isoformat_datetime_separator; -1 only for YYYY-Www- (9 bytes), whose date
// then always fails (its weekday is the NUL).
func (b isoBytes) separator() int {
	n := len(b)
	switch {
	case n == 7:
		return 7
	case b[4] == '-' && b[5] == 'W':
		if n > 8 && b[8] == '-' {
			if n == 9 {
				return -1
			}
			if n > 10 && isDigit(b[10]) {
				return 8
			}
			return 10
		}
		return 8
	case b[4] == '-':
		return 10
	case b[4] == 'W':
		idx := 7
		for idx < n && isDigit(b[idx]) {
			idx++
		}
		if idx < 9 {
			return idx
		}
		if idx%2 == 0 {
			return 7
		}
		return 8
	}
	return 8
}

// date is parse_isoformat_date over the first end bytes (it may read past them, as C does).
func (b isoBytes) date(end int) (y, m, d int, ok bool) {
	p, ok := b.digits(0, &y, 4)
	if !ok {
		return
	}
	sep := b.at(p) == '-'
	if sep {
		p++
	}
	if b.at(p) == 'W' {
		week, day := 0, 0
		if p, ok = b.digits(p+1, &week, 2); !ok {
			return
		}
		if p < end {
			if sep {
				if b.at(p) != '-' {
					return 0, 0, 0, false
				}
				p++
			}
			if _, ok = b.digits(p, &day, 1); !ok {
				return
			}
		} else {
			day = 1
		}
		return isoToYMD(y, week, day)
	}
	if p, ok = b.digits(p, &m, 2); !ok {
		return
	}
	if sep {
		if b.at(p) != '-' {
			return 0, 0, 0, false
		}
		p++
	}
	_, ok = b.digits(p, &d, 2)
	return
}

func isLeap(y int) bool { return y%4 == 0 && (y%100 != 0 || y%400 == 0) }

func daysInMonth(y, m int) int {
	if m == 2 && isLeap(y) {
		return 29
	}
	return [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[m-1]
}

// ordinal is ymd_to_ord: days since 0001-01-01, which is day 1.
func ordinal(y, m, d int) int {
	n := y - 1
	days := n*365 + n/4 - n/100 + n/400
	for k := 1; k < m; k++ {
		days += daysInMonth(y, k)
	}
	return days + d
}

// isoToYMD is iso_to_ymd: an ISO year, week and weekday as a proleptic Gregorian date.
func isoToYMD(year, week, day int) (int, int, int, bool) {
	if year < 1 || year > 9999 {
		return 0, 0, 0, false
	}
	first := ordinal(year, 1, 1)
	weekday := (first + 6) % 7 // Monday is 0
	if week <= 0 || week >= 53 {
		if week != 53 || !(weekday == 3 || weekday == 2 && isLeap(year)) {
			return 0, 0, 0, false
		}
	}
	if day <= 0 || day >= 8 {
		return 0, 0, 0, false
	}
	monday := first - weekday
	if weekday > 3 {
		monday += 7
	}
	t := time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, monday+(week-1)*7+day-2)
	return t.Year(), int(t.Month()), t.Day(), true
}
