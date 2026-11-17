// Package writeops holds the write tools of mcp-n4-kit/mcp_n4/tools_write.py (217-1240) as
// plan.Impl values; the plan engine runs them. This slice ports n4_set_slot and
// n4_invoke_action.
package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// SetSlot is n4_set_slot: one slot, a Status slot always whole (value and status).
var SetSlot = plan.Impl{ScopeOrds: ordArg, Plan: setSlotPlan, Readback: setSlotReadback}

// setSlotData carries what the read-back compares: the requested value and, for baja:Simple, the
// type the slot must hold afterwards.
type setSlotData struct {
	requested any
	wireType  string
}

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
	simpleType := str(args, "simple_type")
	if _, given := args.Get("simple_type"); given && valueType != "baja:Simple" {
		return nil, plan.ToolError("simple_type only applies to baja:Simple")
	}
	if simpleType != "" && !typeSpec.MatchString(simpleType) {
		return nil, plan.ToolError("simple_type must be a type spec module:Type, got " + pyjson.StrRepr(simpleType))
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
	if nodes, err = plan.Resolve(c, nodes, slot); err != nil {
		return nil, err
	}
	node := box.Node(nodes, slot)
	if node == nil {
		return nil, plan.ToolError(fmt.Sprintf(msgSetSlotMissing, pyjson.StrRepr(slot), ordText))
	}
	t, _ := node.Get("t")
	slotType := pyjson.Str(t)
	notes := []any{}
	wireType := ""
	if valueType == "baja:Simple" {
		if wireType, err = simpleWireType(nodes, slot, slotType, simpleType); err != nil {
			return nil, err
		}
		if wireType != slotType {
			notes = append(notes, "the value type "+slotType+" is replaced by "+wireType+
				"; the rollback restores "+slotType)
		}
	} else if slotType != valueType {
		return nil, plan.ToolError("slot " + pyjson.StrRepr(slot) + " is " + slotType + ", not " + valueType)
	}
	previous, err := plan.Observe(nodes, slot, valueType)
	if err != nil {
		return nil, err
	}
	links := incomingLinks(nodes, slot)
	if note := incomingNote(links, slot); note != "" {
		notes = append(notes, note)
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	op, inverse := plan.ToBSON(valueType, requested), plan.ToBSON(valueType, previous)
	if valueType == "baja:Simple" {
		op, inverse = simpleBSON(wireType, requested), simpleBSON(slotType, previous)
	}
	return &plan.Planned{
		Ops:     []any{setOp(h, slot, op)},
		Inverse: []any{setOp(h, slot, inverse)},
		Notes:   notes, Targets: targets, Keys: obj("incoming_links", links), Data: setSlotData{requested, wireType}}, nil
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
	if nodes, err = plan.Resolve(c, nodes, slot); err != nil {
		return nil, nil, nil, "", err
	}
	observed, err := plan.Observe(nodes, slot, valueType)
	if err != nil {
		return nil, nil, nil, "", err
	}
	data := p.Data.(setSlotData)
	v := verdict(observed, data.requested)
	if data.wireType != "" {
		if n := box.Node(nodes, slot); n != nil {
			if t, _ := n.Get("t"); pyjson.Str(t) != data.wireType {
				v = "mismatch"
			}
		}
	}
	return data.requested, replies[0], observed, v, nil
}

func verdict(observed, requested any) string {
	if box.PyEqual(observed, requested) {
		return "verified"
	}
	return "mismatch"
}
