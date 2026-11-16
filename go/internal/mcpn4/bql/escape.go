package bql

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ordescape"
)

// The station decodes $xx / $uxxxx in an /ord/ path before it parses the ORD and its BQL, so a
// value is checked as the station will read it: decoded (ordescape), then refused when any
// decoded stage holds a character that could end a quoted literal, split a clause or end the
// bql part (retro 2026-11-16 M2 verifier: source_glob nomatch$27$20or$20source$20like$20$27*
// matched every alarm record live).

// Characters a value may not decode to: inside a BQL literal (alarm filters), and in an ORD
// path (a history id, where only the ORD separator matters; a space or quote there is data).
const (
	literalForbidden = "'\" |"
	pathForbidden    = "|"
)

// Messages of the escape check, as data.
const (
	escapeRule    = "%s must not hold a $ escape the station decodes to %s (it decodes $xx and $uxxxx before it reads the query): got %q"
	malformedRule = "%s holds a malformed $ escape ('$' must start $xx or $uxxxx with hex digits; the station refuses it): got %q"
)

// checkEscapes is nil when value, decoded as the station decodes it, holds no character of
// forbid and no control character.
func checkEscapes(arg, value, forbid string) error {
	return checkEscapesShown(arg, value, value, forbid)
}

// checkEscapesShown is checkEscapes on value naming shown, the caller's text it came from.
func checkEscapesShown(arg, value, shown, forbid string) error {
	r, bad, err := ordescape.Forbidden(value, forbid)
	switch {
	case err != nil:
		return valueError(fmt.Sprintf(malformedRule, arg, shown))
	case bad:
		return valueError(fmt.Sprintf(escapeRule, arg, ordescape.Describe(r), shown))
	}
	return nil
}
