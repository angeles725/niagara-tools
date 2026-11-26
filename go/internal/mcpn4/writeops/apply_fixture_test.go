package writeops_test

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The golden deployed instance of n4_apply (#934): Gold (h10) holds templateConfig (h11) with the
// exposed slots setpoint, label and gain, bound by B1 and B2 to the component Pt (h13) (slots limit
// and tag) and by B3 to Pt/Inner (h14) (slot gain); a deployed instance names each target by handle.
const (
	apParent = "station:|slot:/Lab/Units"
	apGold   = apParent + "/Gold"
)

func apLeaf(name, typ, v string) *pyjson.Object {
	return obj("nm", "p", "n", name, "t", typ, "v", v)
}

func apNode(name, typ, h string, kids ...any) *pyjson.Object {
	o := obj("nm", "p", "n", name, "t", typ)
	if h != "" {
		o.Set("h", h)
	}
	o.Set("s", kids)
	return o
}

// apState is the value of every slot of an instance: the bindings' targets (b1, b2, b3), the
// config slots (setpoint, label, cfgGain) and the bound slots (limit, tag, gain).
type apState struct{ b1, b2, b3, setpoint, label, cfgGain, limit, tag, gain string }

func apGolden() apState {
	return apState{"h:13", "h:13", "h:14", "20.0", "x", "1.0", "20.0", "", "1.0"}
}

func (s apState) inner() *pyjson.Object {
	return apNode("Inner", "baja:Folder", "14", apLeaf("gain", "baja:Double", s.gain))
}

func (s apState) pt() *pyjson.Object {
	return apNode("Pt", "kitControl:Limiter", "13", apLeaf("limit", "baja:Double", s.limit),
		apLeaf("tag", "baja:String", s.tag), s.inner())
}

func (s apState) tree() *pyjson.Object {
	b := func(n, src, to, slot string) *pyjson.Object {
		return apNode(n, "template:ConfigBinding", "", apLeaf("sourceSlot", "baja:String", src),
			apLeaf("targetOrd", "baja:Ord", to), apLeaf("targetSlot", "baja:String", slot))
	}
	cfg := apNode("templateConfig", "template:TemplateConfig", "11",
		apLeaf("setpoint", "baja:Double", s.setpoint), apLeaf("label", "baja:String", s.label),
		apLeaf("gain", "baja:Double", s.cfgGain),
		b("B1", "setpoint", s.b1, "limit"), b("B2", "label", s.b2, "tag"), b("B3", "gain", s.b3, "gain"))
	return obj("nm", "p", "h", "10", "t", "baja:Folder", "s", []any{cfg, s.pt()})
}

// apFlat is the load of a tree: every node by its path, as LoadTree returns it.
func apFlat(tree *pyjson.Object) *pyjson.Object {
	out := obj()
	var put func(path string, n *pyjson.Object)
	put = func(path string, n *pyjson.Object) {
		out.Set(path, n)
		kids, _ := get(n, "s").([]any)
		for _, k := range kids {
			kid := k.(*pyjson.Object)
			p := get(kid, "n").(string)
			if path != "" {
				p = path + "/" + p
			}
			put(p, kid)
		}
	}
	put("", tree)
	return out
}

// apTypes is the registry channel's answer: every type of the instance, none declaring a slot.
func apTypes() *pyjson.Object {
	t := obj()
	for _, spec := range []string{"baja:Folder", "kitControl:Limiter", "template:TemplateConfig", "template:ConfigBinding"} {
		t.Set(spec, obj("c", []any{}, "p", "baja:Component"))
	}
	t.Set("baja:Component", obj())
	return t
}

// apStation is the station before the call: the parent with Gold (and any other child named) and
// the golden instance as it loads.
func apStation(gold apState, others ...string) *typed {
	return apStationOf(gold.tree(), gold.pt(), others...)
}

func apStationOf(tree, pt *pyjson.Object, others ...string) *typed {
	parent := obj("", obj("h", "5", "t", "baja:Folder"), "Gold", obj("h", "10", "t", "baja:Folder"))
	for _, n := range others {
		parent.Set(n, obj("h", "6", "t", "baja:Folder"))
	}
	return &typed{loads: loads{nodes: obj(apParent, parent, apGold, apFlat(tree), apGold+"/Pt", apFlat(pt))},
		types: apTypes()}
}

func apArgs(rows ...any) *pyjson.Object {
	return obj("parent_ord", apParent, "golden", "Gold", "rows", rows)
}

func apRow(name string, configs ...any) *pyjson.Object {
	return obj("name", name, "configs", obj(configs...))
}

// apKid is the child of node named n; nil when absent.
func apKid(node *pyjson.Object, n string) *pyjson.Object {
	kids, _ := get(node, "s").([]any)
	for _, k := range kids {
		if o := k.(*pyjson.Object); get(o, "n") == n {
			return o
		}
	}
	return nil
}

// apBindingOrds is the targetOrd of each ConfigBinding of an add op's body, by binding name.
func apBindingOrds(op any) map[string]any {
	out := map[string]any{}
	cfg := apKid(get(op.(*pyjson.Object), "b").(*pyjson.Object), "templateConfig")
	kids, _ := get(cfg, "s").([]any)
	for _, kid := range kids {
		k := kid.(*pyjson.Object)
		if get(k, "t") == "template:ConfigBinding" {
			out[get(k, "n").(string)] = get(apKid(k, "targetOrd"), "v")
		}
	}
	return out
}
