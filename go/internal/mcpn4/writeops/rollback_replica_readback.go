package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// rollbackReadbackCopies is rollbackReadbackAll plus, for the rollback of an in-doubt replication,
// each copy's outcome in observed.copies: a found copy reads back "removed" or "still_present";
// any copy the rollback kept (changed, unverified, not sent) caps a verified verdict at partial.
func rollbackReadbackCopies(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	requested, accepted, observed, verdict, err := rollbackReadbackAll(c, args, p, replies, inverse)
	d := p.Data.(*rollbackData)
	if err != nil || d.copies == nil {
		return requested, accepted, observed, verdict, err
	}
	loaded, out, kept := map[string]*pyjson.Object{}, []any{}, false
	for _, e := range d.copies {
		entry := asObj(e)
		state := pyjson.Str(get(entry, "state"))
		if state == copyFound {
			where := pyjson.Str(get(entry, "target"))
			nodes, ok := loaded[where]
			if !ok {
				if nodes, err = c.LoadTree(where, 1, box.LoadOptions{}); err != nil {
					return fail(err)
				}
				loaded[where] = nodes
			}
			state = "removed"
			if n, _ := get(entry, "name").(string); box.Node(nodes, n) != nil {
				state = "still_present"
			}
		}
		kept = kept || state == copyChanged || state == copyUnverified || state == copyNotSent
		out = append(out, obj("op", get(entry, "op"), "target", get(entry, "target"), "name", get(entry, "name"), "state", state))
	}
	asObj(observed).Set("copies", out)
	if kept && verdict == "verified" {
		verdict = "partial"
	}
	return requested, accepted, observed, verdict, nil
}
