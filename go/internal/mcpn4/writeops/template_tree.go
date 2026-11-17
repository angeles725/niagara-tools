package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// tplConfigType is the template's configuration component (template-rt BTemplateConfig): its
// bindings name their targets by template handle (targetOrd h:758), so it is never deployed.
const tplConfigType = "template:TemplateConfig"

// tplLink is a link inside the template: the path of the component holding it and its slots.
type tplLink struct{ holder, source, sourceSlot, targetSlot string }

// tplLeftOut is what the body of a deploy does not carry, by path below the template component
// ("" the component itself): links, ORD values naming a template handle, passwords, slot facets
// and categories.
type tplLeftOut struct {
	links                               []tplLink
	handleOrds, passwords, facets, cats []string
}

// tplBody is the add body of base (no name: the op names it), without each node tplLeftOut
// lists and without the template configuration; paths maps each template handle to its path,
// the source a link names.
func tplBody(base *tplNode, paths map[string]string) (*pyjson.Object, *tplLeftOut) {
	out := &tplLeftOut{}
	return tplNodeBody(base, "", paths, out), out
}

func tplNodeBody(n *tplNode, path string, paths map[string]string, out *tplLeftOut) *pyjson.Object {
	body := obj("nm", "p")
	if n.typ != "" {
		body.Set("t", n.typ)
	}
	if path != "" {
		body.Set("n", n.name)
	}
	if n.hasValue {
		body.Set("v", n.value)
	}
	if n.flags != "" {
		body.Set("f", n.flags)
	}
	if n.facets {
		out.facets = append(out.facets, tplLabel(path))
	}
	if n.categories {
		out.cats = append(out.cats, tplLabel(path))
	}
	kids := []any{}
	for _, k := range n.kids {
		p := joinPath(path, k.name)
		switch {
		case k.typ == tplConfigType:
			continue
		case k.typ == "baja:Link" || k.typ == "baja:ConversionLink":
			out.links = append(out.links, linkOf(k, path, paths))
			continue
		case k.typ == "baja:Ord" && strings.HasPrefix(k.value, "h:"):
			out.handleOrds = append(out.handleOrds, p+" ("+k.value+")")
			continue
		case k.typ == "baja:Password":
			out.passwords = append(out.passwords, p)
			continue
		}
		kids = append(kids, tplNodeBody(k, p, paths, out))
	}
	if len(kids) > 0 {
		body.Set("s", kids)
	}
	return body
}

// linkOf reads a link node held by the component at holder; a source named by a template handle
// is shown by its path.
func linkOf(k *tplNode, holder string, paths map[string]string) tplLink {
	l := tplLink{holder: tplLabel(holder)}
	for _, s := range k.kids {
		switch s.name {
		case "sourceOrd":
			l.source = s.value
			if p, ok := paths[strings.TrimPrefix(s.value, "h:")]; ok && strings.HasPrefix(s.value, "h:") {
				l.source = tplLabel(p)
			}
		case "sourceSlotName":
			l.sourceSlot = s.value
		case "targetSlotName":
			l.targetSlot = s.value
		}
	}
	return l
}

// tplHandles maps each handle in base's subtree to its path below base.
func tplHandles(base *tplNode) map[string]string {
	out := map[string]string{}
	var walk func(n *tplNode, path string)
	walk = func(n *tplNode, path string) {
		if n.handle != "" {
			out[n.handle] = path
		}
		for _, k := range n.kids {
			walk(k, joinPath(path, k.name))
		}
	}
	walk(base, "")
	return out
}

// tplAt is the node at path below base; nil when absent.
func tplAt(base *tplNode, path string) *tplNode {
	n := base
	if path == "" {
		return n
	}
	for _, name := range strings.Split(path, "/") {
		if n = n.child(name); n == nil {
			return nil
		}
	}
	return n
}

// tplLabel is a path for a note: "(root)" for the template component itself.
func tplLabel(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

// tplTypes are the type specs of every node the body adds, each once in tree order, with the
// path of the first node of that type.
func tplTypes(body *pyjson.Object) (specs []string, where map[string]string) {
	where = map[string]string{}
	var walk func(o *pyjson.Object, path string)
	walk = func(o *pyjson.Object, path string) {
		if t, ok := get(o, "t").(string); ok {
			if _, seen := where[t]; !seen {
				where[t] = tplLabel(path)
				specs = append(specs, t)
			}
		}
		for _, k := range objects(get(o, "s")) {
			walk(k, joinPath(path, name(k)))
		}
	}
	walk(body, "")
	return specs, where
}
