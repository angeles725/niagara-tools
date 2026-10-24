package px

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyxml"
)

func strip(s string) string { return strings.TrimFunc(s, pyjson.IsSpace) }

// Parse is parse_px: the summary of one .px document, or the ValueError that refuses it (a
// DOCTYPE or ENTITY declaration before parsing, malformed XML, a root other than <px>).
func Parse(text string, maxItems int) (*pyjson.Object, error) {
	if declaration(text) {
		return nil, &pyjson.ValueError{Msg: "refusing a Px document with a DOCTYPE or ENTITY declaration"}
	}
	root, err := pyxml.Parse(text)
	var pe *pyxml.Error
	if errors.As(err, &pe) {
		return nil, &pyjson.ValueError{Msg: "not well-formed Px XML: " + pe.Error()}
	}
	if err != nil {
		return nil, err
	}
	if root.Tag != "px" {
		return nil, &pyjson.ValueError{Msg: "not a Px document: the root element is <" + root.Tag + ">, expected <px>"}
	}
	imports := newBounded(maxItems)
	for _, el := range root.Iter() {
		if name, _ := el.Get("name"); el.Tag == "module" && name != "" {
			imports.add(name)
		}
	}
	scope := root
	for _, c := range root.Children {
		if c.Tag == "content" {
			scope = c
			break
		}
	}
	widgets, bindings, bound := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	canvases, layers, images, included := newBounded(maxItems), newBounded(maxItems), newBounded(maxItems), newBounded(maxItems)
	labels, others, charts := newBounded(maxItems), newBounded(maxItems), newBounded(maxItems)
	boundCut := false
	for _, el := range scope.Iter()[1:] {
		tag := el.Tag
		switch {
		case strings.HasSuffix(tag, "Binding"):
			count(bindings, tag)
		case tag == "layer" || tag == "Layer":
			if name, _ := el.Get("name"); name != "" {
				layers.add(name)
			}
		default:
			count(widgets, tag)
		}
		if vs, _ := el.Get("viewSize"); vs != "" {
			canvases.add(obj("type", tag, "name", attr(el, "name"), "view_size", vs))
		}
		if layer, _ := el.Get("layer"); layer != "" {
			layers.add(layer)
		}
		if text, ok := el.Get("text"); ok && strip(text) != "" {
			labels.add(text)
		}
		if strings.Contains(tag, "Chart") {
			ords := []any{}
			for _, sub := range el.Iter() {
				for _, a := range sub.Attrs {
					if isOrd(a.Value) {
						ords = append(ords, a.Value)
					}
				}
			}
			charts.add(obj("type", tag, "name", attr(el, "name"), "ords", ords))
		}
		for _, a := range el.Attrs {
			value := strip(a.Value)
			switch {
			case !isOrd(value) && !strings.HasPrefix(value, "module://"):
			case isSlotOrd(value):
				key := BoundComponent(value)
				group, ok := bound.Get(key)
				if !ok && bound.Len() >= maxItems {
					boundCut = true
					continue
				}
				if list, _ := group.([]any); !contains(list, value) {
					bound.Set(key, append(list, value))
				}
			case tag == "PxInclude" || strings.HasPrefix(value, "file:^") && strings.HasSuffix(pyjson.Lower(value), ".px"):
				included.add(value)
			case isImage(value) || a.Name == "image":
				images.add(value)
			default:
				others.add(value)
			}
		}
	}
	var viewSize any
	if len(canvases.items) > 0 {
		viewSize, _ = canvases.items[0].(*pyjson.Object).Get("view_size")
	}
	out := obj("version", attr(root, "version"), "media", attr(root, "media"), "imports", imports.items,
		"canvases", canvases.items, "view_size", viewSize, "layers", layers.items, "widget_counts", widgets,
		"binding_counts", bindings, "bound_components", bound, "images", images.items, "included_px", included.items,
		"charts", charts.items, "other_ords", others.items, "labels", labels.items)
	for _, b := range []struct {
		key  string
		list *bounded
	}{{"imports", imports}, {"canvases", canvases}, {"layers", layers}, {"images", images},
		{"included_px", included}, {"charts", charts}, {"other_ords", others}, {"labels", labels}} {
		if b.list.truncated {
			out.Set(b.key+"_truncated", true)
		}
	}
	if boundCut {
		out.Set("bound_components_truncated", true)
	}
	return out, nil
}

func contains(list []any, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
