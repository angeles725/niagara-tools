package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// commRun is one n4_commissioning_report call: its bounds, the station reads it made and the
// tables it read, each read once whatever the number of sections that use it.
type commRun struct {
	t                              *Tools
	scope, slot                    string // the scope as an ORD and as a slot path
	limit, maxRows, timeout, depth int
	requests, rows, truncated      int // truncated: the rows of the first truncated table
	tables                         map[string]*commTable
	expectHistories, expectAlarms  *expectFilter
	spyderSpec                     []spyderDevice // nil: no spyder_spec given
	spyderHardware                 bool           // spyder_hardware: add the model_mismatch check
}

// commTable is one query's answer: its rows by column name, whether they were cut, or why the
// query failed (then rows is nil: a failed read, never an empty table).
type commTable struct {
	name   string
	rows   []*pyjson.Object
	cut    bool
	failed string
}

// table reads the table called name once. err is only an error that fails the whole call (an
// authentication failure); any other failure is the table's failed text.
func (r *commRun) table(name string) (*commTable, error) {
	if tb, ok := r.tables[name]; ok {
		return tb, nil
	}
	q, tb := commQueries[name], &commTable{name: name}
	base := q.base
	if base == "" {
		base = r.scope
	}
	r.requests++
	columns, raw, cut, _, err := r.t.query(base, q.query, r.maxRows, r.timeout)
	if err == nil {
		tb.rows, err = bql.ByPosition(columns, raw, q.names)
	}
	if err != nil {
		section, err := r.t.sectionError(err)
		if err != nil {
			return nil, err
		}
		tb.rows, tb.failed = nil, fmt.Sprintf(commQueryFailed, name, lookup(section, "error"))
	}
	if tb.cut = cut && tb.failed == ""; tb.cut && r.truncated == 0 {
		r.truncated = max(len(tb.rows), 1)
	}
	r.rows += len(tb.rows)
	r.tables[name] = tb
	return tb, nil
}

// cutWhy is why an absence cannot be asserted from a truncated table.
func (tb *commTable) cutWhy() string {
	return fmt.Sprintf(commCutAbsence, tb.name, len(tb.rows))
}

// section is a new section called name; a truncated table among tables marks it truncated.
func (r *commRun) section(name string, tables ...*commTable) *commSection {
	s := &commSection{name: name, limit: r.limit, hidden: map[string]int{}}
	for _, tb := range tables {
		if tb != nil && tb.cut && s.truncated == 0 {
			s.truncated = max(len(tb.rows), 1)
		}
	}
	return s
}

// failure is err as a not-checked reason, or err itself when it must fail the whole call.
func (r *commRun) failure(what string, err error) (string, error) {
	section, err := r.t.sectionError(err)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s: %s", what, lookup(section, "error")), nil
}

// pointName is how a point row is named: its display name, else its slot name.
func pointName(row *pyjson.Object) string {
	if d := strings.TrimSpace(cellText(row, displayColumn)); d != "" {
		return d
	}
	return cellText(row, "Name")
}

// parentPath is the path of the component holding path ("" at the root).
func parentPath(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}

// lastSegment is the slot name ending path, escaped as the station writes it.
func lastSegment(path string) string { return path[strings.LastIndexByte(path, '/')+1:] }

// ownerOf is the deepest path of set that is an ancestor of path ("" when none is).
func ownerOf[T any](set map[string]T, path string) string {
	for p := parentPath(path); strings.Contains(p, "/"); p = parentPath(p) {
		if _, ok := set[p]; ok {
			return p
		}
	}
	return ""
}
