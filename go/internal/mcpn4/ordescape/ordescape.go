// Package ordescape decodes Niagara's $ escapes exactly as the station does, so a value can be
// checked for what the station will read, not for what the caller typed. The station decodes
// $xx / $uxxxx in an /ord/ path before it parses the ORD and its BQL: a value holding $27 or
// $7c reaches the query as a quote or the ORD separator (live, 4.15.6, retro 2026-11-16 M2
// verifier). Every value a tool composes into an ORD is decoded here, then checked.
package ordescape

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// maxDecodes bounds Decoded: each pass shortens the text, so a fixpoint arrives sooner anyway.
const maxDecodes = 8

// ErrMalformed is an escape the station could not decode (it throws on it).
var ErrMalformed = errors.New("malformed $ escape: '$' must start $xx or $uxxxx with hex digits")

// Unescape is one pass of com.tridium.util.EscUtil.unescape (organized/baja/baja/vineflower/
// com/tridium/util/EscUtil.java:85-118; SlotPath.unescape delegates to it, javax/baja/naming/
// SlotPath.java:135-136): '$' then 'u' and four hex digits is one UTF-16 unit, '$' and two hex
// digits one character; any other text is copied. Hex digits are 0-9, A-F, a-f (nre TextUtil.
// hexCharToInt, TextUtil.java:126-135, which throws on anything else, as a short escape does).
func Unescape(s string) (string, error) {
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
			return "", ErrMalformed
		}
		v := 0
		for _, c := range in[i+1 : i+1+n] {
			d := hexValue(c)
			if d < 0 {
				return "", ErrMalformed
			}
			v = v<<4 | d
		}
		b.WriteRune(rune(v))
		i += n
	}
	return b.String(), nil
}

// Decoded is s after Unescape repeated until it no longer changes: the station decodes once, but a
// decoded '$' (from $24) may be read as an escape again downstream, so every stage is the text a
// reader could see. $2427 decodes to $27, then to a quote.
func Decoded(s string) ([]string, error) {
	stages := []string{s}
	for range maxDecodes {
		next, err := Unescape(stages[len(stages)-1])
		if err != nil {
			return nil, err
		}
		if next == stages[len(stages)-1] {
			return stages, nil
		}
		stages = append(stages, next)
	}
	return stages, nil
}

// Forbidden is the first character of forbid, or any control character, in any decoded stage of
// s; ok is false when there is none. A malformed escape is an error.
func Forbidden(s, forbid string) (rune, bool, error) {
	stages, err := Decoded(s)
	if err != nil {
		return 0, false, err
	}
	for _, stage := range stages {
		for _, r := range stage {
			if unicode.IsControl(r) || strings.ContainsRune(forbid, r) {
				return r, true, nil
			}
		}
	}
	return 0, false, nil
}

// Describe names r for a refusal message.
func Describe(r rune) string {
	if unicode.IsControl(r) || r == ' ' {
		return fmt.Sprintf("U+%04X", r)
	}
	return fmt.Sprintf("%q", r)
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
