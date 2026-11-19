package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// setNullColumnWarnings sets the reply's "warnings" when a projected column is no slot (M1).
func (t *Tools) setNullColumnWarnings(out *pyjson.Object, query string, columns []string, rows []*pyjson.Object) {
	if warnings := t.nullColumnWarnings(query, columns, rows); len(warnings) > 0 {
		out.Set("warnings", strs(warnings))
	}
}

// nullColumnWarnings are n4_bql_query's warnings for projected columns NULL on every row that are
// no slot of the queried type (field retro 2026-11-19 M1). The type contract is read through the
// session's registry cache (loadContracts, once per session); only when some column is NULL on
// every row. A client that cannot read contracts, or a contract that does not describe the whole
// chain, gives the unchecked warning for a name it does not list.
func (t *Tools) nullColumnWarnings(query string, columns []string, rows []*pyjson.Object) []string {
	typ := bql.FromType(query)
	nulls := bql.NullColumns(query, columns, rows)
	if typ == "" || len(nulls) == 0 {
		return nil
	}
	var slots map[string]bool
	why := "the client cannot read type contracts"
	if tl, ok := t.Ctx.Session.Client.(typeLoader); ok {
		types, err := loadContracts(tl, &t.undescribed, []string{typ})
		if err != nil {
			why = t.Ctx.Scrub("the type registry could not be read: " + err.Error())
		} else {
			names, note := declaredProps(types, typ)
			slots, why = map[string]bool{}, note
			for _, n := range names {
				slots[n] = true
			}
		}
	}
	var out []string
	for _, slot := range nulls {
		switch {
		case slots[slot]:
		case slots != nil && why == "":
			out = append(out, bql.NoSlotWarning(slot, typ))
		default:
			out = append(out, bql.UncheckedWarning(slot, typ, why))
		}
	}
	return out
}
