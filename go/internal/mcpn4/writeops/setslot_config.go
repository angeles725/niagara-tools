package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// A deployed template instance's templateConfig copies a changed exposed slot to the target of the
// ConfigBinding whose sourceSlot names it (BTemplateConfig.changed / changeBoundProperty, template-rt);
// a targetSlot "#Name" or "#DisplayName" renames the target instead (changeNameAttribute). #990 M3.
const (
	templateConfigType = "template:TemplateConfig"
	configBindingType  = "template:ConfigBinding"
	// cfgInstanceDepth is how deep below the instance root a binding's target handle is looked for.
	cfgInstanceDepth = 6
)

// configTarget is the slot a templateConfig slot propagates to.
type configTarget struct{ ord, slot string }

func (t *configTarget) object() *pyjson.Object { return obj("ord", t.ord, "slot", t.slot) }

// configTargetOf is the target the slot of the component at ordText propagates to: the first binding whose
// sourceSlot names it, as BTemplateConfig.changed returns after the first match; nil when the
// component is no templateConfig or no binding names the slot. A #Name/#DisplayName binding and a
// target handle not found inside the instance (the templateConfig's parent) are refused.
func configTargetOf(c plan.Client, ordText, slot string, nodes *pyjson.Object) (*configTarget, error) {
	// the plan's load (slotDepth 2 for a top-level slot) reaches each binding's own slots, as the
	// depth-2 load of n4_list_templates does
	if pyjson.Str(get(box.Node(nodes, ""), "t")) != templateConfigType || strings.Contains(slot, "/") {
		return nil, nil
	}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		if pyjson.Str(get(box.Node(kids, name), "t")) != configBindingType {
			continue
		}
		b := box.Children(nodes, name)
		if pyjson.Str(box.ChildValue(b, "sourceSlot", "")) != slot {
			continue
		}
		targetOrd, targetSlot := pyjson.Str(box.ChildValue(b, "targetOrd", "")), pyjson.Str(box.ChildValue(b, "targetSlot", ""))
		if strings.HasPrefix(targetSlot, "#") {
			return nil, plan.ToolError(fmt.Sprintf(msgConfigAttribute, pyjson.StrRepr(slot), targetSlot, targetOrd))
		}
		path, err := instanceHandle(c, ordText, targetOrd)
		if err != nil {
			return nil, err
		}
		return &configTarget{ord: path, slot: targetSlot}, nil
	}
	return nil, nil
}

// instanceHandle is the station ORD of the component whose handle the binding's target ORD names,
// looked for in the instance root (the templateConfig's parent) and below it.
func instanceHandle(c plan.Client, ordText, targetOrd string) (string, error) {
	cut := strings.LastIndexByte(ordText, '/')
	if path, ok := slotOrdInside(targetOrd, ordText, cut); ok || path != "" {
		if !ok {
			return "", plan.ToolError(fmt.Sprintf(msgConfigTarget, targetOrd, ordText[:cut], cfgInstanceDepth))
		}
		return path, nil
	}
	handle := strings.TrimPrefix(targetOrd[strings.LastIndex(targetOrd, "|")+1:], "h:")
	if cut < 0 || !strings.HasPrefix(targetOrd[strings.LastIndex(targetOrd, "|")+1:], "h:") {
		return "", plan.ToolError(fmt.Sprintf(msgConfigTarget, targetOrd, ordText, cfgInstanceDepth))
	}
	root := ordText[:cut]
	nodes, err := c.LoadTree(root, cfgInstanceDepth, box.LoadOptions{})
	if err != nil {
		return "", err
	}
	for _, name := range nodes.Keys() {
		if get(box.Node(nodes, name), "h") == handle {
			if name == "" {
				return root, nil
			}
			return root + "/" + name, nil
		}
	}
	return "", plan.ToolError(fmt.Sprintf(msgConfigTarget, targetOrd, root, cfgInstanceDepth))
}

// configReadback is the verdict of a templateConfig write: the propagated target must hold the
// value too.
func configReadback(c plan.Client, t *configTarget, valueType string, requested any, v string) (string, error) {
	if t == nil || v != "verified" {
		return v, nil
	}
	_, nodes, err := plan.Handle(c, t.ord, slotDepth(t.slot))
	if err != nil {
		return "", err
	}
	observed, err := plan.Observe(nodes, t.slot, valueType)
	if err != nil {
		return "", err
	}
	return verdict(observed, requested), nil
}

// setSlotAnyPlanWith plans either form, then checks every propagated target against the write
// scope: the station writes it although the call names only the templateConfig.
func setSlotAnyPlanWith(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	p, err := setSlotAnyPlan(c, args)
	if err != nil || w == nil || w.Scope == nil {
		return p, err
	}
	plans := []*plan.Planned{p}
	if d, ok := p.Data.(*setSlotBatch); ok {
		plans = d.plans
	}
	for _, q := range plans {
		if t := q.Data.(setSlotData).config; t != nil {
			if err := w.Scope.Check(t.ord); err != nil {
				return nil, plan.ToolError(fmt.Sprintf(msgConfigScope, t.ord, t.slot, err.Error()))
			}
		}
	}
	return p, nil
}

// withKeys is base with the keys of extra added after its own.
func withKeys(base, extra *pyjson.Object) *pyjson.Object {
	for _, k := range extra.Keys() {
		v, _ := extra.Get(k)
		base.Set(k, v)
	}
	return base
}

// slotOrdInside reads a binding target given as a slot ORD (n4_apply re-points a copy's bindings
// so, #934): path is its station ORD ("" when the target is no slot ORD) and ok says it lies in the
// instance root (the templateConfig's parent) or below it.
func slotOrdInside(targetOrd, ordText string, cut int) (path string, ok bool) {
	tail := targetOrd[strings.LastIndex(targetOrd, "|")+1:]
	if cut < 0 || !strings.HasPrefix(tail, "slot:/") {
		return "", false
	}
	path, root := "station:|"+tail, ordText[:cut]
	return path, ord.PlainSlotOrd(path) && (path == root || strings.HasPrefix(path, root+"/"))
}
