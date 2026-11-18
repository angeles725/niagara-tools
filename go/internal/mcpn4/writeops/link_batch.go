package writeops

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The batch form of n4_create_link: each entry is planned, inverted, recovered and read back exactly
// as the single form (linkPlan, linkInverse, linkRecover, linkReadback); the batch concatenates their
// ops and inverses in entry order, so the engine journals and n4_rollback undoes them as one batch.

// linkBatch is a batch plan's private data: each entry and its own plan.
type linkBatch struct {
	entries []*pyjson.Object
	plans   []*plan.Planned
}

// linkBatchPlan plans every entry; one refused entry refuses the batch (its message names the
// entry), and so does a second entry for the link an earlier one makes. The key is the station's
// handles and the slot names, so another spelling of the same ORD (a trailing slash, a slot path
// that loads its component) is the same link.
func linkBatchPlan(c plan.Client, entries []*pyjson.Object) (*plan.Planned, error) {
	n := strconv.Itoa(len(entries))
	out := &plan.Planned{Ops: []any{}, Inverse: []any{}, Targets: pyjson.NewObject(), Writes: len(entries),
		Notes: []any{"creates " + n + " link(s) in one batch under one confirmation token, " + n + " write(s) of " +
			"the budget (one per link; write_budget.remaining shows what is left); n4_rollback removes every " +
			"link and is outside the budget"}}
	d, seen := &linkBatch{entries: entries}, map[string]int{}
	for i, e := range entries {
		p, err := linkPlan(c, e)
		if err != nil {
			return nil, namedError(linkEntryName(i), err)
		}
		ld := p.Data.(linkData)
		key := fmt.Sprintf("%v\x00%s\x00%v\x00%s", ld.srcH, str(e, "source_slot"), ld.tgtH, str(e, "target_slot"))
		if j, dup := seen[key]; dup {
			return nil, plan.ToolError(fmt.Sprintf("%s makes the link %s.%s -> %s.%s twice (%s already does); one entry per link",
				linkEntryName(i), str(e, "source_ord"), str(e, "source_slot"), str(e, "target_ord"), str(e, "target_slot"),
				linkEntryName(j)))
		}
		seen[key] = i
		out.Ops, out.Inverse = append(out.Ops, p.Ops...), append(out.Inverse, p.Inverse...)
		for _, k := range p.Targets.Keys() {
			v, _ := p.Targets.Get(k)
			out.Targets.Set(k, v)
		}
		d.plans = append(d.plans, p)
	}
	out.Data = d
	return out, nil
}

// linkBatchInverse is each sent entry's inverse in entry order (a refused entry adds none); one
// reply that cannot be trusted makes the batch's inverse an error, and Recover takes over. Only the
// entries with a reply count: after a failed op the engine passes the replies of the sent ones.
func linkBatchInverse(d *linkBatch, replies []any) ([]any, error) {
	out := []any{}
	for i := range replies {
		inv, err := linkInverse(d.plans[i], replies[i:i+1])
		if err != nil {
			return nil, namedError(linkEntryName(i), err)
		}
		out = append(out, inv...)
	}
	return out, nil
}

// linkBatchRecover keeps the inverse of every entry whose reply can be trusted and looks for the
// others' links by their slots (linkRecover); observed lists what each entry found.
func linkBatchRecover(c plan.Client, d *linkBatch, replies []any) (any, []any, error) {
	observed, inverse := []any{}, []any{}
	for i := range replies {
		e := d.entries[i]
		inv, err := linkInverse(d.plans[i], replies[i:i+1])
		var found any
		if err != nil {
			if found, inv, err = linkRecover(c, e, d.plans[i], replies[i:i+1]); err != nil {
				return nil, nil, err
			}
		}
		observed = append(observed, linkEntryView(e, "observed", found))
		inverse = append(inverse, inv...)
	}
	return observed, inverse, nil
}

// linkBatchReadback reads back each entry as the single form does: requested lists each entry's
// requested link, observed {source_ord, source_slot, target_ord, target_slot, requested, observed,
// verdict} per entry. The batch takes the entries' verdict when they all agree, else partial (some
// links verified, others missing, changed or refused).
func linkBatchReadback(c plan.Client, d *linkBatch, replies []any) (any, any, any, string, error) {
	requested, observed, verdict := []any{}, []any{}, ""
	for i, e := range d.entries {
		inv, err := linkInverse(d.plans[i], replies[i:i+1])
		if err != nil {
			return fail(err)
		}
		r, _, o, v, err := linkReadback(c, e, d.plans[i], replies[i:i+1], inv)
		if err != nil {
			return fail(err)
		}
		requested = append(requested, r)
		observed = append(observed, linkEntryView(e, "requested", r, "observed", o, "verdict", v))
		switch {
		case verdict == "":
			verdict = v
		case verdict != v:
			verdict = "partial"
		}
	}
	return requested, replies, observed, verdict, nil
}

// linkBatchUnapplied is true when no entry's target has a link it did not have when planned.
func linkBatchUnapplied(c plan.Client, d *linkBatch) (bool, error) {
	for i, e := range d.entries {
		if none, err := linkUnapplied(c, e, d.plans[i]); err != nil || !none {
			return false, err
		}
	}
	return true, nil
}

// linkEntryView is an entry's four arguments followed by kv.
func linkEntryView(e *pyjson.Object, kv ...any) *pyjson.Object {
	out := obj()
	for _, k := range linkKeys {
		out.Set(k, str(e, k))
	}
	for i := 0; i < len(kv); i += 2 {
		out.Set(kv[i].(string), kv[i+1])
	}
	return out
}
