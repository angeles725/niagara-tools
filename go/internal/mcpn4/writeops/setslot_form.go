package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// SetSlot is n4_set_slot: one slot (the single form: ord, slot, value, value_type, status?,
// simple_type?), or 1 to MaxSetSlotWrites of them in writes under one dry run and one token
// (overamps retro row M6, backlog OD040). A Status slot is always written whole (value and status).
// A batch spends one write of --max-writes, as n4_set_tags and n4_add_history do.
var SetSlot = plan.Impl{ScopeCheck: setSlotScope, Plan: setSlotAnyPlan, Readback: setSlotAnyReadback}

// MaxSetSlotWrites bounds the entries of one n4_set_slot batch.
const MaxSetSlotWrites = 50

// singleKeys are the arguments of the single form; singleRequired those it cannot do without, in
// the order the schema listed them as required before the batch form existed.
var (
	singleKeys     = []string{"ord", "slot", "value", "value_type", "status", "simple_type"}
	singleRequired = []string{"ord", "slot", "value", "value_type"}
)

// setSlotWrites are the writes entries; batch is false for the single form. A call that mixes the
// two forms, lacks a required single-form argument, or lists 0 or more than MaxSetSlotWrites
// entries (or a non-object one) is refused.
func setSlotWrites(args *pyjson.Object) (entries []*pyjson.Object, batch bool, err error) {
	raw, batch := args.Get("writes")
	if !batch {
		for _, k := range singleRequired {
			if _, ok := args.Get(k); !ok {
				return nil, false, plan.ToolError("missing required argument: " + k)
			}
		}
		return nil, false, nil
	}
	for _, k := range singleKeys {
		if _, ok := args.Get(k); ok {
			return nil, true, plan.ToolError("n4_set_slot takes either the single form (ord, slot, value, " +
				"value_type, status, simple_type) or writes, not both: got writes and " + k)
		}
	}
	list, _ := raw.([]any)
	if len(list) == 0 || len(list) > MaxSetSlotWrites {
		return nil, true, plan.ToolError("writes must list 1 to 50 entries {ord, slot, value, value_type, status?, simple_type?}")
	}
	for i, v := range list {
		e, ok := v.(*pyjson.Object)
		if !ok {
			return nil, true, plan.ToolError(entryName(i) + " must be an object {ord, slot, value, value_type, status?, simple_type?}")
		}
		entries = append(entries, e)
	}
	return entries, true, nil
}

// setSlotScope checks the form, then that every ORD written (the single ord, or each entry's) is
// under the write scope: one entry outside refuses the whole call.
func setSlotScope(s *ord.Scope, args *pyjson.Object) error {
	entries, batch, err := setSlotWrites(args)
	if err != nil {
		return err
	}
	if !batch {
		return s.Check(str(args, "ord"))
	}
	for _, e := range entries {
		if err := s.Check(str(e, "ord")); err != nil {
			return err
		}
	}
	return nil
}

// setSlotAnyPlan plans either form.
func setSlotAnyPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	entries, batch, err := setSlotWrites(args)
	if err != nil {
		return nil, err
	}
	if !batch {
		return setSlotPlan(c, args)
	}
	return setSlotBatchPlan(c, entries)
}

// setSlotAnyReadback reads back either form.
func setSlotAnyReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	if d, ok := p.Data.(*setSlotBatch); ok {
		return setSlotBatchReadback(c, d, replies)
	}
	return setSlotReadback(c, args, p, replies, inverse)
}
