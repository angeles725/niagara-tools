package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// tplExposed is a setting a deployer fills (an exposed property): the slot it writes (holder
// path below the template component and slot name), the template's value for it and the type of
// the configuration property that holds that value. A manifest setting with a slotPath writes
// that path; one without it writes the target of the template configuration's ConfigBinding
// whose sourceSlot is its name (targetOrd h:<handle>, targetSlot). Any other setting is the
// configuration's own metadata (uID, version, deployed...), not a deploy-time value.
type tplExposed struct {
	tplSetting
	holder, slot, deflt, srcType string
	hasDefault, pseudo           bool
}

// tplExposedSettings resolves the manifest's settings against the template tree, in manifest
// order, and lists the metadata settings by name.
func tplExposedSettings(m *tplManifest, base *tplNode, handles map[string]string) ([]*tplExposed, map[string]bool, error) {
	config, bindings := tplConfig(base)
	var out []*tplExposed
	meta := map[string]bool{}
	for _, s := range m.settings {
		e := &tplExposed{tplSetting: s}
		switch b, bound := bindings[s.name]; {
		case s.slotPath != "":
			i := strings.LastIndex(s.slotPath, "/")
			if i >= 0 {
				e.holder = s.slotPath[:i]
			}
			e.slot = s.slotPath[i+1:]
			if t := tplAt(base, s.slotPath); t != nil {
				e.deflt, e.hasDefault = slotValue(t)
				e.srcType = t.typ
			}
		case bound:
			path, ok := handles[strings.TrimPrefix(b[0], "h:")]
			if !strings.HasPrefix(b[0], "h:") || !ok || inConfig(path, config) {
				return nil, nil, plan.ToolError("template: setting " + pyjson.StrRepr(s.name) +
					" is bound to " + b[0] + ", which is not a component of the template")
			}
			e.holder, e.slot, e.pseudo = path, b[1], strings.HasPrefix(b[1], "#")
			if src := config.child(s.name); src != nil {
				e.deflt, e.hasDefault = slotValue(src)
				e.srcType = src.typ
			}
		default:
			meta[s.name] = true
			continue
		}
		out = append(out, e)
	}
	return out, meta, nil
}

// tplConfig is base's template configuration child (nil when none) and its bindings by
// sourceSlot: [targetOrd, targetSlot].
func tplConfig(base *tplNode) (*tplNode, map[string][2]string) {
	bindings := map[string][2]string{}
	for _, k := range base.kids {
		if k.typ != tplConfigType {
			continue
		}
		for _, b := range k.kids {
			if b.typ != "template:ConfigBinding" {
				continue
			}
			var src, ord, slot string
			for _, f := range b.kids {
				switch f.name {
				case "sourceSlot":
					src = f.value
				case "targetOrd":
					ord = f.value
				case "targetSlot":
					slot = f.value
				}
			}
			bindings[src] = [2]string{ord, slot}
		}
		return k, bindings
	}
	return nil, bindings
}

func inConfig(path string, config *tplNode) bool {
	return config != nil && (path == config.name || strings.HasPrefix(path, config.name+"/"))
}

// slotValue is the value a slot node holds: its own, or its value child's for a Status struct.
func slotValue(n *tplNode) (string, bool) {
	if v := n.child("value"); v != nil && !n.hasValue {
		return v.value, v.hasValue
	}
	return n.value, n.hasValue
}

// tplStatusStruct is a node whose value lives in its value child (StatusNumeric, StatusString...).
func tplStatusStruct(n *tplNode) bool {
	return n.child("value") != nil || strings.HasPrefix(n.typ, "baja:Status") && n.typ != "baja:Status"
}

// setSlotValue writes value into slot node n (its value child for a Status struct).
func setSlotValue(n *tplNode, value string) {
	if !tplStatusStruct(n) {
		n.value, n.hasValue = value, true
		return
	}
	v := n.child("value")
	if v == nil {
		v = &tplNode{name: "value"}
		n.kids = append(n.kids, v)
	}
	v.value, v.hasValue = value, true
}

// fromBSON is a declared default ({nm, t, v, s}) as a template node named name.
func fromBSON(name string, o *pyjson.Object) *tplNode {
	n := &tplNode{name: name}
	n.typ, _ = get(o, "t").(string)
	if v, ok := get(o, "v").(string); ok {
		n.value, n.hasValue = v, true
	}
	for _, k := range objects(get(o, "s")) {
		kn, _ := get(k, "n").(string)
		n.kids = append(n.kids, fromBSON(kn, k))
	}
	return n
}
