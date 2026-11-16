package bql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The query text checks: the ones every base shares, and the select of a table base
// (alarm:, history:). ValidateQuery keeps the station-base grammar in query.go.

var (
	// tableSelectRe is a select over a table base: the alarm and history tables take no
	// 'from <type>' (validateTableQuery refuses one), so where and order by follow the columns.
	tableSelectRe = regexp.MustCompile(`^(?is)select` + space + `+` + nonSpace + `.*$`)
	// fromRe is a from keyword; quotedRe a quoted literal ('' escapes a quote) masked before it.
	fromRe   = regexp.MustCompile(`(?i)(^|` + space + `)from(` + space + `|$)`)
	quotedRe = regexp.MustCompile(`'(?:[^']|'')*'`)
)

// Messages of the table-query checks, as data.
const (
	tableSelectRule = "query over the %s base must be 'select <columns> [where ...] [order by ...]'"
	tableFromRule   = "query over the %s base must not have a from clause (the base is the table): use 'select <columns> [where ...] [order by ...]'"
)

// checkQueryText refuses a stripped query holding '|', a control character, or more than
// MaxQueryLength code points, in its text or in the text BqlQuery.make decodes it to
// (DecodeForBql): the checks every base shares.
func checkQueryText(q string) error {
	d, err := decodedQuery("query", q)
	if err != nil {
		return err
	}
	if err := checkPlainText(d); err != nil {
		return err
	}
	return checkPlainText(q)
}

// checkPlainText is checkQueryText on one form of the text.
func checkPlainText(q string) error {
	switch {
	case strings.Contains(q, "|"):
		return valueError("query must not contain '|': only one select is composed into the ORD")
	case controlRe.MatchString(q):
		return valueError("query must not contain control characters")
	case pyjson.Len(q) > MaxQueryLength:
		return valueError(fmt.Sprintf("query is longer than %d characters", MaxQueryLength))
	}
	return nil
}

// validateTableQuery is ValidateQuery for a table base: the same '|', control and length
// checks, then one select with no from clause (a from inside a quoted literal is data).
func validateTableQuery(query any, base string) (string, error) {
	q, ok := query.(string)
	if !ok {
		return "", valueError(fmt.Sprintf(tableSelectRule, base))
	}
	if err := checkQueryText(Strip(q)); err != nil {
		return "", err
	}
	if q = Strip(q); !tableSelectRe.MatchString(q) {
		return "", valueError(fmt.Sprintf(tableSelectRule, base))
	}
	// Live: history: ignores a from clause and answers the whole history; alarm: answers 404.
	d, _ := DecodeForBql(q) // checkQueryText refused a malformed escape
	if fromRe.MatchString(quotedRe.ReplaceAllString(q, "''")) || fromRe.MatchString(quotedRe.ReplaceAllString(d, "''")) {
		return "", valueError(fmt.Sprintf(tableFromRule, base))
	}
	return q, nil
}
