package readtools

import (
	"math/big"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/capacity"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/license"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// licenseNoPlan is the plan_result note of a licence page that was not read.
const licenseNoPlan = "the licence manager page was not read (see status): whether the plan fits is unknown"

// licenseUsage adds usage (per global capacity rule: used, limit, headroom) to out when the
// page was read, and plan_result when args hold a plan: two BQL selects, no write.
func (t *Tools) licenseUsage(out *pyjson.Object, info *license.Info, args *pyjson.Object, timeout int) {
	planArg, planned := args.Get("plan")
	if info == nil {
		if planned {
			out.Set("plan_result", obj("fits", nil, "over", []any{}, "note", licenseNoPlan))
		}
		return
	}
	g, _ := t.Ctx.Session.Client.(capacity.OrdGetter)
	usages := capacity.Measure(g, info, float64(timeout))
	list := make([]any, len(usages))
	for i, u := range usages {
		entry := obj("rule", u.Rule, "used", count(u.Used), "limit", count(u.Limit), "headroom", count(u.Headroom()),
			"excluded_modules", strs(u.Excluded))
		if u.Note != "" {
			entry.Set("note", u.Note)
		}
		list[i] = entry
	}
	out.Set("usage", list)
	if planned {
		out.Set("plan_result", planResult(usages, planArg))
	}
}

// planResult is fits (null when unknown), over [{rule, by}] and, when something is unknown, a note.
func planResult(usages []capacity.Usage, planArg any) *pyjson.Object {
	add := map[string]int{}
	if p, ok := planArg.(*pyjson.Object); ok {
		for _, r := range capacity.Rules {
			if v, ok := p.Get(r.PlanKey); ok {
				if num, isNum := v.(pyjson.Number); isNum {
					if n, whole := num.BigInt(); whole && n.Cmp(big.NewInt(0)) >= 0 && n.IsInt64() {
						add[r.Name] = int(n.Int64())
					}
				}
			}
		}
	}
	res := capacity.Check(usages, add)
	over := make([]any, len(res.Over))
	for i, o := range res.Over {
		over[i] = obj("rule", o.Rule, "by", o.By)
	}
	var fits any
	if res.Fits != nil {
		fits = *res.Fits
	}
	out := obj("fits", fits, "over", over)
	if len(res.Notes) > 0 {
		out.Set("note", joinNotes(res.Notes))
	}
	return out
}
