package writeops

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ReorderSlots is n4_reorder_slots: the complete new order of one component's dynamic children,
// sent as the station's ReorderOp. Wire format (Niagara 4.14 baja, decompiled corpus
// /home/cristian/niagara-research/organized/baja/baja/decompiled/javax/baja/sync/ReorderOp.java):
// the op id is 111, the char 'o' (ReorderOp.java:30-32, SyncOp.make case 111), and its body is the
// SyncOp's "h" (the parent's handle) plus one attribute "o", the slot names joined with ';'
// (ReorderOp.java:55-66; decode splits on ';', :69-78). The station's commit looks every name up
// as a property and calls BComponent.reorder, which throws unless the list holds exactly every
// dynamic property (ComponentSlotMap.java:2080-2092 under .../com/tridium/sys/schema/: the count
// must equal the dynamic count, a frozen slot is a FrozenSlotException). So the plan refuses
// anything but a permutation of the dynamic children, and the inverse is the previous order.
var ReorderSlots = plan.Impl{ScopeOrds: reorderScope, Plan: reorderPlan, Readback: reorderReadback}

// msgReorderBehaviour is the plan note on why the order matters.
const msgReorderBehaviour = "Slot order is behaviour: point extensions run onExecute in slot order on every " +
	"point cycle. Frozen slots do not move; the inverse restores the previous order."

type reorderData struct {
	parentOrd string
	order     []string
}

func reorderScope(args *pyjson.Object) []any { return []any{str(args, "parent_ord")} }

// reorderOp is the BOX body of a ReorderOp: the names joined with ';' in "o".
func reorderOp(h any, order []string) *pyjson.Object {
	return obj("nm", "o", "h", h, "o", strings.Join(order, ";"))
}

// reorderNames reads the order argument; it does not touch the station.
func reorderNames(args *pyjson.Object) ([]string, error) {
	v, _ := args.Get("order")
	list, _ := v.([]any)
	if len(list) < 1 {
		return nil, plan.ToolError("order needs the name of every dynamic child of parent_ord, got 0")
	}
	out := make([]string, 0, len(list))
	for _, e := range list {
		n, err := plan.ExistingName("order", e)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// dynamicChildren is the names of the children in nodes (a load of the parent) that are dynamic
// slots, in slot order, and the names of those the parent's type declares (frozen). The client
// must read the type contracts: without them a frozen slot cannot be told from a dynamic one.
func dynamicChildren(c plan.Client, nodes *pyjson.Object, parentOrd string) (dynamic, frozen []string, err error) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, nil, plan.ToolError("a reorder needs the station's type definitions (registry channel) to " +
			"tell frozen slots from dynamic ones; this client cannot read them")
	}
	typ, _ := get(box.Node(nodes, ""), "t").(string)
	ct := plan.NewContracts(tl)
	for _, n := range box.Children(nodes, "").Keys() {
		slot, err := ct.Declared(typ, obj("n", n))
		var unknown *plan.UnansweredType
		if errors.As(err, &unknown) {
			return nil, nil, plan.ToolError(unknown.Error() + "; " + parentOrd + " cannot be told from its frozen slots")
		}
		if err != nil {
			return nil, nil, err
		}
		if slot != nil {
			frozen = append(frozen, n)
		} else {
			dynamic = append(dynamic, n)
		}
	}
	return dynamic, frozen, nil
}

// reorderCheck refuses an order that is not a permutation of the dynamic children.
func reorderCheck(order, dynamic, frozen []string, parentOrd string) error {
	seen := map[string]bool{}
	for _, n := range order {
		switch {
		case seen[n]:
			return plan.ToolError(pyjson.StrRepr(n) + " is listed twice in order")
		case slices.Contains(frozen, n):
			return plan.ToolError(pyjson.StrRepr(n) + " is a frozen slot of " + parentOrd +
				" and cannot move; order lists its dynamic children only")
		case !slices.Contains(dynamic, n):
			return plan.ToolError(pyjson.StrRepr(n) + " is not a dynamic child of " + parentOrd)
		}
		seen[n] = true
	}
	var missing []string
	for _, n := range dynamic {
		if !seen[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		return plan.ToolError(fmt.Sprintf("order must list every dynamic child of %s once (%d of %d given); missing: %s",
			parentOrd, len(order), len(dynamic), strings.Join(missing, ", ")))
	}
	if slices.Equal(order, dynamic) {
		return plan.ToolError("the dynamic children of " + parentOrd + " are already in that order")
	}
	return nil
}

func reorderPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	order, err := reorderNames(args)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	h, nodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	dynamic, frozen, err := dynamicChildren(c, nodes, parentOrd)
	if err != nil {
		return nil, err
	}
	if err := reorderCheck(order, dynamic, frozen, parentOrd); err != nil {
		return nil, err
	}
	return &plan.Planned{Ops: []any{reorderOp(h, order)}, Inverse: []any{reorderOp(h, dynamic)},
		Notes: []any{parentOrd + ": " + strings.Join(dynamic, ", ") + " -> " + strings.Join(order, ", "),
			msgReorderBehaviour},
		Targets: obj(parentOrd, h), Data: reorderData{parentOrd, order}}, nil
}

// orderAmong is the names of nodes' children that are in names, in the order the station holds.
func orderAmong(nodes *pyjson.Object, names []string) []string {
	out := []string{}
	for _, n := range box.Children(nodes, "").Keys() {
		if slices.Contains(names, n) {
			out = append(out, n)
		}
	}
	return out
}

// reorderReadback loads the parent again: verified only when its dynamic children, read in the
// order the station holds, are the requested order.
func reorderReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	d := p.Data.(reorderData)
	_, nodes, err := plan.Handle(c, d.parentOrd, 1)
	if err != nil {
		return fail(err)
	}
	held := orderAmong(nodes, d.order)
	verdict := "verified"
	if !slices.Equal(held, d.order) {
		verdict = "mismatch"
	}
	return []any{obj("parent_ord", d.parentOrd, "order", stringsAny(d.order))}, replies,
		[]any{obj("parent_ord", d.parentOrd, "order", stringsAny(held))}, verdict, nil
}

// reorderedBack is the rollback read-back of a ReorderOp: the names it lists are in its order.
func reorderedBack(nodes, op *pyjson.Object) bool {
	list, _ := get(op, "o").(string)
	want := strings.Split(list, ";")
	return slices.Equal(orderAmong(nodes, want), want)
}
