// Package bql holds the pure helpers of projected BQL reads (mcp-n4-kit/mcp_n4/bql.py): the one
// place a BQL ORD is composed, the query and base checks, the ITableToCsv reader and the
// inventory summary. Nothing here touches the network. A str is UTF-8 or WTF-8 (pyjson), and a
// position or length counts code points, as Python's str does.
package bql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Row caps, the response byte cap and the query length cap of bql.py.
const (
	DefaultMaxRows   = 5000
	MaxRowsLimit     = 50000
	MaxResponseBytes = 32 << 20
	MaxQueryLength   = 2000
	View             = "view:file:ITableToCsv"
	StationSlot      = "station:|slot:"
)

// space is a str pattern's \s: every code point str.isspace() is true for.
const space = `[\t-\r\x1c- \x{85}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}]`

// nonSpace is \S.
const nonSpace = `[^\t-\r\x1c- \x{85}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}]`

var (
	selectRe     = regexp.MustCompile(`^(?is)select` + space + `+` + nonSpace + `.*?` + space + `+from` + space + `+` + nonSpace + `+(?:` + space + `+.*)?$`)
	selectListRe = regexp.MustCompile(`^(?is)select` + space + `+(.*?)` + space + `+from` + space)
	controlRe    = regexp.MustCompile(`[\x00-\x1f\x7f\x{feff}]`)
	escapeRe     = regexp.MustCompile(`\$(?:u([0-9a-fA-F]{4})|([0-9a-fA-F]{2}))`)
)

// NameColumns are the headers whose cells are decoded slot names (Slot Path stays escaped:
// it is ORD-ready).
var NameColumns = []string{"Name"}

// LocalDeviceSlots are the frozen slots where a network keeps the station's own device.
var LocalDeviceSlots = []string{"localDevice"}

// strip is str.strip(): leading and trailing str.isspace() code points removed.
func strip(s string) string { return strings.TrimFunc(s, pyjson.IsSpace) }

// UnescapeSlot decodes Niagara slot-name escaping: $20 is a space, $u00e9 is é. A surrogate
// escape decodes to a lone surrogate (WTF-8), as chr() gives one.
func UnescapeSlot(text string) string {
	return escapeRe.ReplaceAllStringFunc(text, func(m string) string {
		n, _ := strconv.ParseUint(strings.TrimPrefix(m[1:], "u"), 16, 32)
		return pyjson.Chr(rune(n))
	})
}

// StripControl drops control characters (and a BOM) from one cell or header.
func StripControl(text string) string { return controlRe.ReplaceAllString(text, "") }

// ValidateQuery returns the stripped query, or a ValueError unless it is one plain select. No
// '|' and no control character anywhere, quoted literals included.
func ValidateQuery(query any) (string, error) {
	q, ok := query.(string)
	if !ok {
		return "", valueError("query must be a string starting with select")
	}
	q = strip(q)
	switch {
	case strings.Contains(q, "|"):
		return "", valueError("query must not contain '|': only one select is composed into the ORD")
	case controlRe.MatchString(q):
		return "", valueError("query must not contain control characters")
	case pyjson.Len(q) > MaxQueryLength:
		return "", valueError(fmt.Sprintf("query is longer than %d characters", MaxQueryLength))
	case !selectRe.MatchString(q):
		return "", valueError("only 'select <columns> from <type> [where ...]' is allowed")
	}
	return q, nil
}

// BaseSlot is base_slot: the slot path (/Drivers, / for the root) of a base given as an ORD or
// a path.
func BaseSlot(base any) (string, error) {
	b, ok := base.(string)
	if !ok || b == "" {
		return "", valueError("base must be a station slot ORD such as station:|slot:/Drivers")
	}
	path := b
	if strings.HasPrefix(b, StationSlot) {
		path = b[len(StationSlot):]
	} else if strings.HasPrefix(b, "slot:") {
		path = b[len("slot:"):]
	}
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "|") || strings.Contains(path, "//") || controlRe.MatchString(path) {
		return "", valueError("base must be a plain slot path such as station:|slot:/Drivers, got " + pyjson.StrRepr(b))
	}
	if path = strings.TrimRight(path, "/"); path == "" {
		path = "/"
	}
	return path, nil
}

// ComposeOrd is compose_ord: the single ORD of query over base. The base is checked first.
func ComposeOrd(base, query any) (string, error) {
	slot, err := BaseSlot(base)
	if err != nil {
		return "", err
	}
	q, err := ValidateQuery(query)
	if err != nil {
		return "", err
	}
	return StationSlot + slot + "|bql:" + q + "|" + View, nil
}

// SelectedSlots is selected_slots: the queried slot names in order, nil for '*' (or an empty
// name). The CSV headers are display names a localized station translates; the select list is
// ours, so it names the columns by position.
func SelectedSlots(query string) []string {
	m := selectListRe.FindStringSubmatch(strip(query))
	if m == nil {
		return nil
	}
	var names []string
	for _, part := range strings.Split(m[1], ",") {
		n := strip(part)
		if n == "*" || n == "" {
			return nil
		}
		names = append(names, n)
	}
	return names
}

// UniqueColumns makes header names unique: a repeated Type becomes Type#2, Type#3, ...
func UniqueColumns(header []string) []string {
	seen := map[string]bool{}
	columns := make([]string, 0, len(header))
	for _, name := range header {
		unique := name
		for n := 2; seen[unique]; n++ {
			unique = fmt.Sprintf("%s#%d", name, n)
		}
		seen[unique] = true
		columns = append(columns, unique)
	}
	return columns
}

// ByPosition is by_position: rows re-keyed by names, one per leading column. No rows map to
// none; fewer columns than names is a ValueError.
func ByPosition(columns []string, rows []*pyjson.Object, names []string) ([]*pyjson.Object, error) {
	if len(rows) == 0 {
		return []*pyjson.Object{}, nil
	}
	if len(columns) < len(names) {
		return nil, valueError(fmt.Sprintf("the BQL answer has %d column(s), expected at least %d (%s)",
			len(columns), len(names), strings.Join(names, ", ")))
	}
	out := make([]*pyjson.Object, 0, len(rows))
	for _, row := range rows {
		o := pyjson.NewObject()
		for i, name := range names {
			v, ok := row.Get(columns[i])
			if !ok {
				v = ""
			}
			o.Set(name, v)
		}
		out = append(out, o)
	}
	return out, nil
}

// CompleteRecords is complete_records: text up to the end of its last complete CSV record. A
// newline ends a record only outside quotes, which quote parity tells ("" is an escaped quote).
func CompleteRecords(text string) string {
	quotes, pos := strings.Count(text, `"`), len(text)
	for {
		nl := strings.LastIndexByte(text[:pos], '\n')
		if nl < 0 {
			return ""
		}
		quotes -= strings.Count(text[nl:pos], `"`)
		if quotes%2 == 0 {
			return text[:nl+1]
		}
		pos = nl
	}
}

func valueError(msg string) error { return &pyjson.ValueError{Msg: msg} }
