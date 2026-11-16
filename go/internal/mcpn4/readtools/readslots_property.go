package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A BOX load answers with the nearest component (live retro row M3): an ORD naming a property of
// a component (a struct such as alarm:AlarmTimestamps, a Status value, a simple) loaded that
// component, so n4_read_slots listed the component's slots. property finds such a property in
// the load and reads it in place of the component.

// propertyRead is the property ord names: the component's ORD (base), the property's path under
// it and its type; nodes are the property's subtree rebased so the property is "", with d its
// fields' declared defaults; self is the property's own entry, listed when it has no fields.
type propertyRead struct {
	base, path, kind string
	nodes            *pyjson.Object
	d                *declared
	self             *pyjson.Object
	elapsed          int
	atDefault        bool // the property is a declared default: every field is its default
}

// propertyPath splits ord into the ORD of the loaded component and the property's path under it
// ("" when ord names the loaded component). The path is the shortest run of trailing segments
// whose every node is loaded (or a declared default) without a handle: a component has one.
func propertyPath(ord string, nodes *pyjson.Object) (base, path string) {
	head, slot, ok := strings.Cut(ord, "slot:")
	if !ok || strings.Contains(slot, "|") {
		return "", ""
	}
	var segs []string
	for _, s := range strings.Split(slot, "/") {
		if s != "" {
			segs = append(segs, s)
		}
	}
	for k := 1; k <= len(segs); k++ {
		tail := segs[len(segs)-k:]
		if handleless(nodes, tail) {
			return head + "slot:/" + strings.Join(segs[:len(segs)-k], "/"), strings.Join(tail, "/")
		}
	}
	return "", ""
}

func handleless(nodes *pyjson.Object, tail []string) bool {
	for i := range tail {
		n := box.Node(nodes, strings.Join(tail[:i+1], "/"))
		if n == nil {
			return false
		}
		if h, _ := n.Get("h"); pyjson.Truthy(h) {
			return false
		}
	}
	return true
}

func rootHandle(nodes *pyjson.Object) any {
	if root := box.Node(nodes, ""); root != nil {
		h, _ := root.Get("h")
		return h
	}
	return nil
}

// property is the property ord names, nil when ord names the loaded component. A candidate is
// confirmed by loading its component's ORD, deep enough for the property's fields and their
// Status children: the same handle must answer (else ord named the component, which happens to
// hold a struct named like the ORD's tail). A failed confirming load reads the component.
func (t *Tools) property(ord string, loaded *pyjson.Object, d *declared) (*propertyRead, error) {
	base, path := propertyPath(ord, d.nodes)
	if path == "" {
		return nil, nil
	}
	nodes, elapsed, err := t.timedLoad(base, 2+strings.Count(path, "/")+1)
	if err != nil || !pyjson.Truthy(rootHandle(loaded)) || !box.PyEqual(rootHandle(nodes), rootHandle(loaded)) {
		return nil, nil
	}
	bd := t.withDefaults(nodes, []string{""})
	if box.Node(bd.nodes, path) == nil {
		return nil, nil
	}
	p := &propertyRead{base: base, path: path, kind: nodeType(bd.nodes, path), nodes: rebase(bd.nodes, path),
		elapsed: elapsed}
	p.d = t.withDefaults(p.nodes, []string{""})
	name := path[strings.LastIndex(path, "/")+1:]
	if bd.paths[path] {
		p.self, err = bd.defaultSlot(path, name)
	} else if p.self, err = slotEntry(bd.nodes, path); err == nil {
		p.self.Set("name", name)
	}
	if err != nil {
		return nil, err
	}
	p.atDefault = bd.paths[path]
	return p, nil
}

// rebase is the subtree of nodes at path, keyed under it ("" for path itself).
func rebase(nodes *pyjson.Object, path string) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range nodes.Keys() {
		v, _ := nodes.Get(k)
		if k == path {
			out.Set("", v)
		} else if rest, ok := strings.CutPrefix(k, path+"/"); ok {
			out.Set(rest, v)
		}
	}
	return out
}

// describe adds to out the component's ORD, the property's path and type, and the decoded value
// of a type the reader knows (alarm:AlarmTimestamps).
func (p *propertyRead) describe(out *pyjson.Object, slots []any) {
	out.Set("component_ord", p.base)
	out.Set("property", p.path)
	out.Set("property_type", p.kind)
	if p.kind == alarmTimestampsType {
		out.Set("alarm_timestamps", alarmTimestamps(slots))
	}
}
