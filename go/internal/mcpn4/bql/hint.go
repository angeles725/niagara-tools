package bql

import (
	"fmt"
	"regexp"
	"strings"
)

// quotedEscape is a quoted literal holding a $xx or $uxxxx escape.
var quotedEscape = regexp.MustCompile(`'([^']*\$(?:u[0-9A-Fa-f]{4}|[0-9A-Fa-f]{2})[^']*)'`)

// EscapeHint explains an empty result of a query whose quoted literal holds a $ escape: BqlQuery
// decodes it inside the query text (escape.go), while slot names are stored escaped, so
// `name = 'Numeric$20Writable'` compares against "Numeric Writable" and matches nothing (field retro
// 2026-11-16 row U6). It returns "" when no literal holds an escape.
func EscapeHint(query string) string {
	m := quotedEscape.FindStringSubmatch(query)
	if m == nil {
		return ""
	}
	decoded, err := DecodeForBql(m[1])
	if err != nil {
		return ""
	}
	return fmt.Sprintf("no rows: '%s' is read by BQL as '%s' (BqlQuery decodes $xx in the query text, while slot "+
		"names are stored escaped); to match the escaped name write '%s' ($24 is a literal $), or use like with *",
		m[1], decoded, strings.ReplaceAll(m[1], "$", "$24"))
}
