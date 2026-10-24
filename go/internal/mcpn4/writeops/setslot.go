// Package writeops holds the write tools of mcp-n4-kit/mcp_n4/tools_write.py (217-1240) as
// plan.Impl values; the plan engine runs them. This slice ports n4_set_slot and
// n4_invoke_action.
package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// SetSlot is n4_set_slot: one slot, a Status slot always whole (value and status).
var SetSlot = plan.Impl{ScopeOrds: ordArg, Plan: setSlotPlan, Readback: setSlotReadback}

type setSlotData struct{ requested any }

func ordArg(args *pyjson.Object) []any { return []any{str(args, "ord")} }

// str is a schema-validated string argument.
func str(args *pyjson.Object, k string) string {
	v, _ := args.Get(k)
	s, _ := v.(string)
	return s
}

func setSlotPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	slot, valueType, ordText := str(args, "slot"), str(args, "value_type"), str(args, "ord")
	if slot == "" {
		return nil, plan.ToolError("slot must be a non-empty slot name")
	}
	parts := strings.Split(slot, "/")
	if last := parts[len(parts)-1]; len(parts) > 1 && (last == "value" || last == "status") {
		return nil, plan.ToolError("refusing to write " + pyjson.StrRepr(slot) + " on its own: " +
			safety.ReasonPartialStatus + " (value and status together) because a value-only write leaves the " +
			"status null (B1199 section 1199.4); pass the Status slot itself with value_type " +
			"baja:StatusNumeric or baja:StatusBoolean")
	}
	for _, part := range parts {
		if _, err := plan.Name("slot", part); err != nil {
			return nil, err
		}
	}
	if !known(valueType) {
		return nil, plan.ToolError(safety.ReasonValueType + " " + strings.Join(plan.ValueTypes, ", "))
	}
	status, hasStatus := args.Get("status")
	if hasStatus && !plan.IsStatusType(valueType) {
		return nil, plan.ToolError("status only applies to baja:StatusNumeric / baja:StatusBoolean")
	}
	if !hasStatus {
		status = "0"
	}
	value, _ := args.Get("value")
	requested, err := plan.Coerce(valueType, value, status)
	if err != nil {
		return nil, err
	}
	h, nodes, err := plan.Handle(c, ordText, slotDepth(slot))
	if err != nil {
		return nil, err
	}
	node := box.Node(nodes, slot)
	if node == nil {
		return nil, plan.ToolError("slot " + pyjson.StrRepr(slot) + " not found on " + ordText + ". The station " +
			"omits a slot whose value equals its type default (0.0, false, an empty string), so n4_set_slot " +
			"cannot read its type and refuses. Check the name with n4_read_slots; if the slot exists at its " +
			"default, set it once in Workbench (or through an action such as `set`) and plan again")
	}
	if t, _ := node.Get("t"); t != valueType {
		return nil, plan.ToolError("slot " + pyjson.StrRepr(slot) + " is " + pyjson.Str(t) + ", not " + valueType)
	}
	previous, err := plan.Observe(nodes, slot, valueType)
	if err != nil {
		return nil, err
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	return &plan.Planned{
		Ops:     []any{setOp(h, slot, plan.ToBSON(valueType, requested))},
		Inverse: []any{setOp(h, slot, plan.ToBSON(valueType, previous))},
		Notes:   []any{}, Targets: targets, Data: setSlotData{requested}}, nil
}

func known(valueType string) bool {
	for _, t := range plan.ValueTypes {
		if t == valueType {
			return true
		}
	}
	return false
}

func setOp(h any, slot string, b *pyjson.Object) *pyjson.Object {
	o := pyjson.NewObject()
	o.Set("nm", "s")
	o.Set("h", h)
	o.Set("n", slot)
	o.Set("b", b)
	return o
}

// slotDepth is _slot_depth: the load depth that reaches the value/status children of slot,
// nested too (audit F3).
func slotDepth(slot string) int { return len(strings.Split(slot, "/")) + 1 }

func setSlotReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	slot, valueType := str(args, "slot"), str(args, "value_type")
	_, nodes, err := plan.Handle(c, str(args, "ord"), slotDepth(slot))
	if err != nil {
		return nil, nil, nil, "", err
	}
	observed, err := plan.Observe(nodes, slot, valueType)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested := p.Data.(setSlotData).requested
	return requested, replies[0], observed, verdict(observed, requested), nil
}

func verdict(observed, requested any) string {
	if box.PyEqual(observed, requested) {
		return "verified"
	}
	return "mismatch"
}
