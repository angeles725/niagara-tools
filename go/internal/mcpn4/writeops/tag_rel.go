package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// tagRel is one requested relation: a tag group (relation n:tagGroup to the group's definition,
// flag r as Workbench's Edit Tags dialog adds it) or a relation id to an endpoint component.
// source is the sourceOrd written; match the sourceOrd forms an existing relation may hold.
type tagRel struct {
	label, id, source string
	match             []string
	group             bool
	implies           []string
}

// same is whether two requests name one relation: the same id to the same endpoint.
func (r tagRel) same(o tagRel) bool { return r.id == o.id && r.source == o.source }

// relBody is the BOX body the add sends.
func (r tagRel) relBody() *pyjson.Object { return tags.RelationBody(r.id, r.source) }

// relFlags are the slot flags of the added relation.
func (r tagRel) relFlags() string {
	if r.group {
		return "r"
	}
	return ""
}

// relSlot is the slot of nodes holding the outbound relation r, "" when none does.
func relSlot(nodes *pyjson.Object, r tagRel) string {
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		if get(box.Node(kids, name), "t") != tags.RelationType {
			continue
		}
		fields := map[string]any{}
		for _, f := range []string{"relationId", "sourceOrd", "inbound"} {
			fields[f] = get(box.Node(nodes, name+"/"+f), "v")
		}
		rel := tags.RelationFields(name, fields)
		if rel.Inbound || rel.ID != r.id {
			continue
		}
		for _, m := range r.match {
			if rel.Endpoint == m {
				return name
			}
		}
	}
	return ""
}

// tagRelAddOp adds r to a target that lacks it (inverse: remove the slot the station names,
// filled in from its reply by setTagsInverse); one already there is refused.
func tagRelAddOp(h any, nodes *pyjson.Object, r tagRel, o string) (op, inv *pyjson.Object, err error) {
	if s := relSlot(nodes, r); s != "" {
		return nil, nil, plan.ToolError(r.label + " on " + o + " is already there (slot " + s + "): refusing")
	}
	op = obj("nm", "a", "h", h, "n", "r?", "b", r.relBody())
	if f := r.relFlags(); f != "" {
		op.Set("f", f)
	}
	return op, obj("nm", "v", "h", h, "n", "r?"), nil
}

// relFieldTypes are the types of a relation's fields, for a field the load sends without one.
var relFieldTypes = map[string]string{"relationId": "baja:String", "sourceOrd": "baja:Ord", "inbound": "baja:Boolean"}

// storedRelation is the body of relation slot s as the station holds it: every field with its
// stored value (the endpoint ORD in the form it was written, any other field kept), without the
// handles and display text a load adds.
func storedRelation(nodes *pyjson.Object, s string) *pyjson.Object {
	fields := []any{}
	kids := box.Children(nodes, s)
	for _, n := range kids.Keys() {
		k := box.Node(kids, n)
		f := obj("nm", "p", "n", n)
		if t, ok := get(k, "t").(string); ok {
			f.Set("t", t)
		} else if t, ok := relFieldTypes[n]; ok {
			f.Set("t", t)
		}
		f.Set("v", get(k, "v"))
		fields = append(fields, f)
	}
	return obj("nm", "p", "t", get(box.Node(nodes, s), "t"), "s", fields)
}

// tagRelRemoveOp removes the relation slot holding r; the inverse re-adds it under its name with
// its flags and stored fields.
func tagRelRemoveOp(h any, nodes *pyjson.Object, r tagRel, o string) (op, inv *pyjson.Object, err error) {
	s := relSlot(nodes, r)
	if s == "" {
		return nil, nil, plan.ToolError(o + " has no " + r.label + " (n4_read_tags lists its tag groups and relations)")
	}
	inv = obj("nm", "a", "h", h, "n", s, "b", storedRelation(nodes, s))
	if f, _ := get(box.Node(nodes, s), "f").(string); f != "" {
		inv.Set("f", f)
	}
	return obj("nm", "v", "h", h, "n", s), inv, nil
}
