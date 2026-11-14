package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// SetFlags is n4_set_flags: the complete flag set of one slot of a component, sent as the
// station's SetFlagsOp (javax.baja.sync.SetFlagsOp, id 'f'). The inverse sets the flags the slot
// held; the read-back loads the slot again and compares the sets.
var SetFlags = plan.Impl{ScopeOrds: ordArg, Plan: setFlagsPlan, Readback: setFlagsReadback}

// flagsDepth loads a component with its direct slots: flags belong to a slot of the component.
const flagsDepth = 1

// setFlagsData is what the read-back compares with: the requested flags, in table order.
type setFlagsData struct{ requested string }

func setFlagsPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	ordText, flags := str(args, "ord"), str(args, "flags")
	slot, err := plan.ExistingName("slot", str(args, "slot"))
	if err != nil {
		return nil, err
	}
	if err := checkFlags(flags); err != nil {
		return nil, err
	}
	requested := canonicalFlags(flags)
	h, nodes, err := plan.Handle(c, ordText, flagsDepth)
	if err != nil {
		return nil, err
	}
	current, err := currentFlags(c, nodes, ordText, slot)
	if err != nil {
		return nil, err
	}
	if sameFlags(current, requested) {
		return nil, plan.ToolError(flagsNothingToDoMsg(slot, ordText, current))
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	return &plan.Planned{
		Ops:     []any{flagsOp(h, slot, requested)},
		Inverse: []any{flagsOp(h, slot, current)}, // the station's own string: restored exactly
		Notes:   []any{flagsChangeNote(slot, ordText, current, requested), flagsUnprovenNote},
		Targets: targets, Data: setFlagsData{requested}}, nil
}

// flagsOp is the BOX body of a SetFlagsOp: the component's handle, the slot name and the whole
// flag set as letters.
func flagsOp(h any, slot, flags string) *pyjson.Object {
	return obj("nm", "f", "h", h, "n", slot, "f", flags)
}

// setFlagsReadback loads the slot again: verified only when the flags it holds are the requested
// set, whatever the station answered to the op.
func setFlagsReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	ordText := str(args, "ord")
	_, nodes, err := plan.Handle(c, ordText, flagsDepth)
	if err != nil {
		return fail(err)
	}
	observed, err := currentFlags(c, nodes, ordText, str(args, "slot"))
	if err != nil {
		return fail(err)
	}
	requested := p.Data.(setFlagsData).requested
	verdict := "mismatch"
	if sameFlags(observed, requested) {
		verdict = "verified"
	}
	return requested, replies[0], observed, verdict, nil
}
