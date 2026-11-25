package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// tagEntry is one component of n4_read_tags.
func (t *Tools) tagEntry(ord string, implied bool, relations string) (*pyjson.Object, error) {
	nodes, err := t.Ctx.Session.Client.(Client).LoadTree(ord, 1, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	return t.tagEntryOf(ord, nodes, implied, relations)
}

// treeLoader is a client that can load several ORDs in one batch (box.Client.LoadMany).
type treeLoader interface {
	LoadMany(ords []string, depth int, opt box.LoadOptions) ([]box.LoadResult, error)
}

// loadTags is the depth-1 load of every ord: one loadSlots frame per box.MaxFrameMessages ords when
// the client can batch, else one LoadTree each. An ord's own failure is its Err.
func (t *Tools) loadTags(ords []string) ([]box.LoadResult, error) {
	client := t.Ctx.Session.Client
	if l, ok := client.(treeLoader); ok {
		return l.LoadMany(ords, 1, box.LoadOptions{})
	}
	out := make([]box.LoadResult, len(ords))
	for i, o := range ords {
		out[i].Nodes, out[i].Err = client.(Client).LoadTree(o, 1, box.LoadOptions{})
	}
	return out, nil
}

// tagEntryOf is tagEntry on the load of ord already done.
func (t *Tools) tagEntryOf(ord string, nodes *pyjson.Object, implied bool, relations string) (*pyjson.Object, error) {
	root := box.Node(nodes, "")
	h, _ := root.Get("h")
	direct, groups, rels := []any{}, []any{}, []any{}
	for _, name := range box.Children(nodes, "").Keys() {
		node := box.Node(nodes, name)
		nm, _ := lookup(node, "nm").(string)
		flags, _ := lookup(node, "f").(string)
		if nm == "" {
			nm = "p" // a test tree may omit it; a station load always sends it
		}
		if lookup(node, "t") == tags.RelationType {
			r := relationOf(nodes, name)
			if r.IsTagGroup() {
				groups = append(groups, obj("group_ord", r.Endpoint, "slot", r.Slot))
			} else {
				rels = append(rels, obj("id", r.ID, "endpoint_ord", r.Endpoint, "inbound", r.Inbound, "slot", r.Slot))
			}
			continue
		}
		if !tags.IsTagSlot(nm, name, flags, pyjson.Truthy(lookup(node, "h"))) {
			continue
		}
		id := tags.FromSlot(name)
		ns, short := "", id
		if p, ok := tags.ParseID(id); ok {
			ns, short = p.Namespace, p.Name
		}
		direct = append(direct, obj("id", id, "namespace", ns, "name", short, "type", lookup(node, "t"),
			"value", lookup(node, "v"), "slot", name, "flags", flags))
	}
	out := obj("ord", ord, "handle", h, "direct", direct, "tag_groups", groups)
	if relations != "none" {
		out.Set("relations", obj("direct", rels))
	}
	if !implied && relations != "all" {
		return out, nil
	}
	caller, ok := t.Ctx.Session.Client.(sessionCaller)
	if !ok {
		out.Set("implied", nil)
		out.Set("implied_unavailable", msgImpliedUnavailable)
		return out, nil
	}
	if implied {
		list, err := impliedTags(caller, h)
		if err != nil {
			return nil, err
		}
		out.Set("implied", list)
	}
	if relations == "all" {
		list, err := impliedRelations(caller, h)
		if err != nil {
			return nil, err
		}
		lookup(out, "relations").(*pyjson.Object).Set("implied", list)
	}
	return out, nil
}

// relationOf is the relation slot name of a load, its fields read from its children.
func relationOf(nodes *pyjson.Object, name string) tags.Relation {
	fields := map[string]any{}
	kids := box.Children(nodes, name)
	for _, k := range kids.Keys() {
		v, _ := kids.Get(k)
		if n, ok := v.(*pyjson.Object); ok {
			fields[k] = lookup(n, "v")
		}
	}
	return tags.RelationFields(name, fields)
}
