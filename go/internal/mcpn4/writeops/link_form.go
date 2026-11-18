package writeops

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// CreateLink is n4_create_link: one checkLinks call that makes the link (the single form:
// source_ord, source_slot, target_ord, target_slot), or 1 to MaxLinks of them in links under one dry
// run and one token (field retro delta U11); the inverse removes each link the station named.
// A reply that cannot be trusted is in-doubt, and Recover looks for each new link by its slots. A
// batch spends one write of --max-writes per link.
var CreateLink = plan.Impl{ScopeCheck: linkAnyScope, Plan: linkAnyPlan, PlanWith: linkAnyPlanWith,
	Inverse: linkAnyInverse, Recover: linkAnyRecover, Readback: linkAnyReadback, Unapplied: linkAnyUnapplied}

// MaxLinks bounds the entries of one n4_create_link batch.
const MaxLinks = 50

// linkKeys are the single form's arguments, all required, and the keys of a links entry.
var linkKeys = []string{"source_ord", "source_slot", "target_ord", "target_slot"}

func linkEntryName(i int) string { return fmt.Sprintf("links[%d]", i) }

// linkEntries are the links entries; batch is false for the single form. A call that mixes the two
// forms, lacks an argument (single form or entry), or lists 0 or more than MaxLinks entries (or a
// non-object one) is refused.
func linkEntries(args *pyjson.Object) (entries []*pyjson.Object, batch bool, err error) {
	raw, batch := args.Get("links")
	if !batch {
		return nil, false, missingLinkArg("", args)
	}
	for _, k := range linkKeys {
		if _, ok := args.Get(k); ok {
			return nil, true, plan.ToolError("n4_create_link takes either the single form (source_ord, source_slot, " +
				"target_ord, target_slot) or links, not both: got links and " + k)
		}
	}
	list, _ := raw.([]any)
	if len(list) == 0 || len(list) > MaxLinks {
		return nil, true, plan.ToolError("links must list 1 to 50 entries {source_ord, source_slot, target_ord, target_slot}")
	}
	for i, v := range list {
		e, ok := v.(*pyjson.Object)
		if !ok {
			return nil, true, plan.ToolError(linkEntryName(i) + " must be an object {source_ord, source_slot, target_ord, target_slot}")
		}
		if err := missingLinkArg(linkEntryName(i)+": ", e); err != nil {
			return nil, true, err
		}
		entries = append(entries, e)
	}
	return entries, true, nil
}

// missingLinkArg refuses the first link argument args lacks, its message prefixed by prefix.
func missingLinkArg(prefix string, args *pyjson.Object) error {
	for _, k := range linkKeys {
		if _, ok := args.Get(k); !ok {
			return plan.ToolError(prefix + "missing required argument: " + k)
		}
	}
	return nil
}

// linkAnyScope checks the form, then each link's scope as the single form does (linkScope): one
// entry outside refuses the whole call, its message naming the entry.
func linkAnyScope(s *ord.Scope, args *pyjson.Object) error {
	entries, batch, err := linkEntries(args)
	if err != nil {
		return err
	}
	if !batch {
		return linkScope(s, args)
	}
	for i, e := range entries {
		if err := linkScope(s, e); err != nil {
			var se *safety.Error
			if errors.As(err, &se) {
				return &safety.Error{Msg: linkEntryName(i) + ": " + se.Msg}
			}
			return err
		}
	}
	return nil
}

// linkAnyPlan plans either form.
func linkAnyPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	entries, batch, err := linkEntries(args)
	if err != nil {
		return nil, err
	}
	if !batch {
		return linkPlan(c, args)
	}
	return linkBatchPlan(c, entries)
}

// linkAnyPlanWith is linkAnyPlan, refusing a batch the remaining write budget cannot hold.
func linkAnyPlanWith(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	p, err := linkAnyPlan(c, args)
	if err != nil {
		return nil, err
	}
	if _, batch := p.Data.(*linkBatch); batch {
		if err := w.CheckRoom(p.Writes, "this link batch"); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func linkAnyInverse(p *plan.Planned, replies []any) ([]any, error) {
	if d, ok := p.Data.(*linkBatch); ok {
		return linkBatchInverse(d, replies)
	}
	return linkInverse(p, replies)
}

func linkAnyRecover(c plan.Client, args *pyjson.Object, p *plan.Planned, replies []any) (any, []any, error) {
	if d, ok := p.Data.(*linkBatch); ok {
		return linkBatchRecover(c, d, replies)
	}
	return linkRecover(c, args, p, replies)
}

func linkAnyReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	if d, ok := p.Data.(*linkBatch); ok {
		return linkBatchReadback(c, d, replies)
	}
	return linkReadback(c, args, p, replies, inverse)
}

func linkAnyUnapplied(c plan.Client, args *pyjson.Object, p *plan.Planned) (bool, error) {
	if d, ok := p.Data.(*linkBatch); ok {
		return linkBatchUnapplied(c, d)
	}
	return linkUnapplied(c, args, p)
}
