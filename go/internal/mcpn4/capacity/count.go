package capacity

import (
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
)

// OrdGetter runs a BQL ORD (the route of n4_bql_query): the CSV text and whether it was cut.
type OrdGetter interface {
	GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error)
}

// Measure is every rule on the licence info, each counted with one select (none when its
// exclusions are unknown). g nil counts nothing: every used is unknown.
func Measure(g OrdGetter, info *license.Info, timeout float64) []Usage {
	out := make([]Usage, 0, len(Rules))
	for _, r := range Rules {
		out = append(out, MeasureRule(g, info, r, timeout))
	}
	return out
}

// MeasureRule is one rule on the licence info: its term and its count.
func MeasureRule(g OrdGetter, info *license.Info, r Rule, timeout float64) Usage {
	term := TermOf(info, r)
	u := Usage{Rule: r.Name, Limit: term.Limit, Unlimited: term.Unlimited, Excluded: term.Excluded, Note: term.Note}
	if term.ExcludedKnown {
		var note string
		u.Used, note = count(g, r, term.Excluded, timeout)
		u.Note = joinNote(u.Note, note)
	}
	return u
}

// count is r's components whose type's module is not excluded, nil and a note when unknown.
func count(g OrdGetter, r Rule, excluded []string, timeout float64) (*int, string) {
	if g == nil {
		return nil, fmt.Sprintf(noteCountFailed, r.Query, noteNoClient)
	}
	ord, err := bql.ComposeOrd(base, r.Query)
	if err == nil {
		var text string
		var cut bool
		if text, cut, err = g.GetOrd(ord, timeout, bql.MaxResponseBytes); err == nil {
			return countTypes(text, cut, excluded)
		}
	}
	return nil, fmt.Sprintf(noteCountFailed, r.Query, err.Error())
}

// countTypes counts the rows of a one-column type table whose module is not excluded.
func countTypes(text string, cut bool, excluded []string) (*int, string) {
	if cut {
		return nil, noteCountCut
	}
	columns, rows, capped, err := bql.ParseCSV(text, bql.MaxRowsLimit, nil)
	if err != nil || len(columns) < 1 {
		return nil, noteCountShape
	}
	if capped {
		return nil, noteCountCut
	}
	n := 0
	for _, row := range rows {
		v, _ := row.Get(columns[0])
		typ, _ := v.(string)
		module, _, _ := strings.Cut(strings.TrimSpace(typ), ":")
		if !slices.Contains(excluded, module) {
			n++
		}
	}
	return &n, ""
}
