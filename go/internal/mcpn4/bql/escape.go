package bql

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Niagara decodes $xx / $uxxxx escapes only inside a BQL body: BqlQuery.make runs
// SlotPath.unescape over the whole BQL text (organized/ bql-rt BqlQuery.java:24); ORD paths,
// history ids and file ORDs are matched escaped (SlotPath.java:229-306) and the /ord/ filter only
// URL-decodes (web-rt OrdTargetFilter.java:44-60,120-133). So operator text that goes into BQL is
// checked on its DECODED form (live 4.15.6: alarm_class=x$27$20or$20alarmClass$20like$20$27$2a
// matched 27 of 27 records), and ORD paths are never decoded before a scope check.

// errMalformed is an escape BqlQuery.make cannot decode (it throws).
var errMalformed = errors.New("malformed $ escape")

// DecodeForBql is s as BqlQuery.make reads it: one pass of com.tridium.util.EscUtil.unescape
// (baja EscUtil.java:86-117): '$' 'u' and four hex digits is one UTF-16 unit, '$' and two hex
// digits one character, any other text is copied; hex is 0-9, A-F, a-f (nre TextUtil.
// hexCharToInt, which throws on anything else, as a short escape does). One pass only: $2427 is
// the literal text $27, exactly as the station reads it.
func DecodeForBql(s string) (string, error) {
	if !strings.Contains(s, "$") {
		return s, nil
	}
	in := []rune(s)
	var b strings.Builder
	for i := 0; i < len(in); i++ {
		if in[i] != '$' {
			b.WriteRune(in[i])
			continue
		}
		n := 2
		if i+1 < len(in) && in[i+1] == 'u' {
			n, i = 4, i+1
		}
		if i+n >= len(in) {
			return "", errMalformed
		}
		v := 0
		for _, c := range in[i+1 : i+1+n] {
			d := hexValue(c)
			if d < 0 {
				return "", errMalformed
			}
			v = v<<4 | d
		}
		b.WriteRune(rune(v))
		i += n
	}
	return b.String(), nil
}

func hexValue(c rune) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	}
	return -1
}

// literalForbidden are the characters a filter value may not decode to inside a BQL literal: a
// quote ends it, a space or '*' widens it (like), '|' and controls are never data here.
const literalForbidden = "'\" *|"

// Messages of the decoded-value checks, as data.
const (
	escapeRule    = "%s must not hold a $ escape that BQL decodes to %s (BqlQuery decodes $xx and $uxxxx in the query text): got %q"
	malformedRule = "%s holds a malformed $ escape ('$' must start $xx or $uxxxx with hex digits; BqlQuery refuses it): got %q"
)

// checkLiteral is nil when value, decoded as BQL decodes it, holds no character of
// literalForbidden and no control character; shown is the caller's text named in a refusal.
func checkLiteral(arg, value, shown string) error {
	decoded, err := DecodeForBql(value)
	if err != nil {
		return valueError(fmt.Sprintf(malformedRule, arg, shown))
	}
	for _, r := range decoded {
		if unicode.IsControl(r) || strings.ContainsRune(literalForbidden, r) {
			return valueError(fmt.Sprintf(escapeRule, arg, describe(r), shown))
		}
	}
	return nil
}

// describe names r for a refusal message.
func describe(r rune) string {
	if unicode.IsControl(r) || r == ' ' {
		return fmt.Sprintf("U+%04X", r)
	}
	return fmt.Sprintf("%q", r)
}

// decodedQuery is the query text as BqlQuery.make reads it, or a refusal naming arg.
func decodedQuery(arg, q string) (string, error) {
	d, err := DecodeForBql(q)
	if err != nil {
		return "", valueError(fmt.Sprintf(malformedRule, arg, q))
	}
	return d, nil
}
