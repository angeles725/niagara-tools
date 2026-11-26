package writeops

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// applyBinding is one ConfigBinding of the golden's templateConfig: the config slot it copies
// (source), the slot it writes (slot) and the path inside the instance of the component that holds
// it (rel; "" is the instance itself). A deployed instance names the target by handle.
type applyBinding struct{ name, source, slot, rel string }

// applyGolden is what the plan reads of the golden instance: its bindings in slot order, the first
// binding naming each config slot (the one BTemplateConfig.changed follows), the templateConfig
// node (the slots' types) and the load by path.
type applyGolden struct {
	ord      string
	bindings []*applyBinding
	bySlot   map[string]*applyBinding
	cfg      *pyjson.Object
	flat     *pyjson.Object
}

// readGolden reads the bindings of the golden instance root (loaded under goldenOrd). Every
// binding must name its target by a handle found inside the instance, or the copy would keep
// pointing at the golden; a configured slot whose first binding renames its target is refused.
func readGolden(goldenOrd, golden string, root *pyjson.Object, configured map[string]bool) (*applyGolden, error) {
	cfg := kidNamed(root, "templateConfig")
	if cfg == nil || get(cfg, "t") != templateConfigType {
		return nil, plan.ToolError(pyjson.StrRepr(golden) + " is not a deployed template instance: it has no " +
			"templateConfig child of type " + templateConfigType)
	}
	g := &applyGolden{ord: goldenOrd, bySlot: map[string]*applyBinding{}, cfg: cfg, flat: obj()}
	handles := map[string]string{}
	var walk func(path string, n *pyjson.Object)
	walk = func(path string, n *pyjson.Object) {
		g.flat.Set(path, n)
		if h, ok := get(n, "h").(string); ok {
			handles[h] = path
		}
		for _, k := range objects(get(n, "s")) {
			walk(joinPath(path, name(k)), k)
		}
	}
	walk("", root)
	for _, b := range objects(get(cfg, "s")) {
		if get(b, "t") != configBindingType {
			continue
		}
		val := func(n string) string { return pyjson.Str(get(kidNamed(b, n), "v")) }
		target, slot := val("targetOrd"), val("targetSlot")
		rel, found := handles[strings.TrimPrefix(target, "h:")]
		if !strings.HasPrefix(target, "h:") || !found {
			return nil, plan.ToolError(fmt.Sprintf(msgConfigTarget, target, goldenOrd, cfgInstanceDepth))
		}
		bind := &applyBinding{name(b), val("sourceSlot"), slot, rel}
		g.bindings = append(g.bindings, bind)
		if g.bySlot[bind.source] == nil {
			g.bySlot[bind.source] = bind
		}
	}
	for _, s := range g.slotNames() {
		if b := g.bySlot[s]; configured[s] && strings.HasPrefix(b.slot, "#") {
			at, err := b.targetOrd(goldenOrd)
			if err != nil {
				return nil, err
			}
			return nil, plan.ToolError(fmt.Sprintf(msgConfigAttribute, pyjson.StrRepr(s), b.slot, at))
		}
	}
	return g, nil
}

// slotNames are the config slots some binding names, sorted.
func (g *applyGolden) slotNames() []string {
	out := []string{}
	for s := range g.bySlot {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// targetOrd is the slot ORD of the component binding b targets inside the instance at base.
func (b *applyBinding) targetOrd(base string) (string, error) { return ord.JoinOrd(base, b.rel) }

// goldenValue is the value of binding b's target slot in the golden, typed as valueType.
func (g *applyGolden) goldenValue(b *applyBinding, valueType string) (any, error) {
	return plan.Observe(g.flat, joinPath(b.rel, b.slot), valueType)
}
