package tags

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// NeqlScheme is the ORD scheme of a NEQL query; NeqlOrdPrefix is how a station names each match
// in the query's Ord column (measured on 4.15: local:|station:|slot:/Folder/Pt).
const (
	NeqlScheme    = "neql:"
	NeqlOrdPrefix = "local:|"
)

// ValidateNeql is the stripped NEQL query, or a ValueError: never empty, no '|' (the ORD
// separator: the station would answer HTTP 400 or read the rest as another scheme), no control
// character, at most bql.MaxQueryLength characters.
func ValidateNeql(query any) (string, error) {
	q, ok := query.(string)
	if !ok {
		return "", &pyjson.ValueError{Msg: "neql must be a string"}
	}
	q = bql.Strip(q)
	switch {
	case q == "":
		return "", &pyjson.ValueError{Msg: "neql must not be empty (e.g. n:point, or hs:equip and n:type = \"baja:Folder\")"}
	case strings.Contains(q, "|"):
		return "", &pyjson.ValueError{Msg: "neql must not contain '|': it separates ORD schemes; append a select with bql instead"}
	case bql.StripControl(q) != q:
		return "", &pyjson.ValueError{Msg: "neql must not contain control characters"}
	case pyjson.Len(q) > bql.MaxQueryLength:
		return "", &pyjson.ValueError{Msg: "neql is longer than 2000 characters"}
	}
	return q, nil
}

// MatchOrd is the station ORD of one Ord cell of a NEQL answer (local:| removed).
func MatchOrd(cell string) string { return strings.TrimPrefix(cell, NeqlOrdPrefix) }

// ValidateSelect is the stripped select appended to a NEQL query, or a ValueError: it runs on the
// matches, so it names columns only (select slotPath, out.value), never a from clause.
func ValidateSelect(query any) (string, error) {
	q, ok := query.(string)
	if !ok {
		return "", &pyjson.ValueError{Msg: "bql must be a string such as select slotPath, out.value"}
	}
	q = bql.Strip(q)
	lower := strings.ToLower(q)
	switch {
	case !strings.HasPrefix(lower, "select "):
		return "", &pyjson.ValueError{Msg: "bql must be one select of columns, e.g. select slotPath, out.value"}
	case strings.Contains(lower, " from "):
		return "", &pyjson.ValueError{Msg: "bql runs on the NEQL matches: give its columns only, no from clause"}
	case strings.Contains(q, "|"), bql.StripControl(q) != q:
		return "", &pyjson.ValueError{Msg: "bql must not contain '|' or control characters"}
	case pyjson.Len(q) > bql.MaxQueryLength:
		return "", &pyjson.ValueError{Msg: "bql is longer than 2000 characters"}
	}
	return q, nil
}
