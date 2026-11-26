package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// applyRow is one row of the table: the copy's name and the config slots it sets, in the order
// the call gave them.
type applyRow struct {
	name    string
	configs *pyjson.Object
}

// applyRows validates rows: 1 to maxCopies entries, each a distinct plain name with an optional
// configs object ({config slot: value}).
func applyRows(args *pyjson.Object) ([]applyRow, error) {
	raw, _ := args.Get("rows")
	list, _ := raw.([]any)
	if len(list) == 0 || len(list) > maxCopies {
		return nil, plan.ToolError("rows must list 1 to " + strconv.Itoa(maxCopies) + " entries")
	}
	out, seen := []applyRow{}, map[string]bool{}
	for _, v := range list {
		entry, ok := v.(*pyjson.Object)
		if !ok {
			return nil, plan.ToolError("each row must be an object {name, configs}")
		}
		n, err := plan.Name("row name", get(entry, "name"))
		if err != nil {
			return nil, err
		}
		if seen[n] {
			return nil, plan.ToolError("a row named " + pyjson.StrRepr(n) + " is given twice")
		}
		seen[n] = true
		configs := obj()
		if c, given := entry.Get("configs"); given {
			if configs, ok = c.(*pyjson.Object); !ok {
				return nil, plan.ToolError("configs of row " + pyjson.StrRepr(n) + " must be an object {config slot: value}")
			}
		}
		out = append(out, applyRow{n, configs})
	}
	return out, nil
}
