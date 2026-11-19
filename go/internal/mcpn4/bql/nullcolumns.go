package bql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A projected name BQL cannot resolve on a component answers "NULL", the same text as a slot
// whose value is null (field retro 2026-11-19 M1: maxOverrideDuration, a ControlPoint facet key,
// read as 135 rows of "NULL"). A column NULL on every row is checked against the queried type's
// contract; one that names no slot of it is warned, never left as a silent "NULL".

// fromTypeRe is the queried type spec of a select (module:Type).
var fromTypeRe = regexp.MustCompile(`(?is)\sfrom\s+([A-Za-z0-9_]+:[A-Za-z0-9_]+)`)

// pseudoColumns are BQL names that are no slot of any type (lower case).
var pseudoColumns = map[string]bool{"slotpath": true, "name": true, "displayname": true, "type": true,
	"parent": true, "handle": true, "navord": true, "ord": true, "path": true, "typespec": true}

// FromType is the type a select queries, "" when it names none (an alarm: or history: select).
func FromType(query string) string {
	m := fromTypeRe.FindStringSubmatch(Strip(query))
	if m == nil {
		return ""
	}
	return m[1]
}

// NullColumns are the projected slot names (first segment of a dotted path) whose column is
// NULL on every row; none without rows, or when the projection does not map one name per column.
func NullColumns(query string, columns []string, rows []*pyjson.Object) []string {
	names := SelectedSlots(query)
	if len(rows) == 0 || len(names) != len(columns) {
		return nil
	}
	var out []string
	for i, col := range columns {
		slot := strings.SplitN(names[i], ".", 2)[0]
		if pseudoColumns[strings.ToLower(slot)] || !allNull(rows, col) {
			continue
		}
		out = append(out, slot)
	}
	return out
}

func allNull(rows []*pyjson.Object, col string) bool {
	for _, r := range rows {
		v, _ := r.Get(col)
		if s, _ := v.(string); !strings.EqualFold(s, "NULL") {
			return false
		}
	}
	return true
}

// NoSlotWarning is the warning of a NULL column that names no slot of typ.
func NoSlotWarning(slot, typ string) string {
	return fmt.Sprintf("%s: NULL on every row and no such slot on %s (facet?): BQL answers NULL for a name it "+
		"cannot resolve; a facet key is read from the facets column", slot, typ)
}

// UncheckedWarning is the warning of a NULL column whose type contract could not be read.
func UncheckedWarning(slot, typ, why string) string {
	return fmt.Sprintf("%s: NULL on every row, and whether it is a slot of %s could not be checked (%s); "+
		"BQL answers NULL for a name it cannot resolve (facet?)", slot, typ, why)
}
