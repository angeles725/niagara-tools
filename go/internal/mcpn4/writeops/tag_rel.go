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
// filled in from its reply by setTagsInverse); one already there is skipped with a note.
func tagRelAddOp(h any, nodes *pyjson.Object, r tagRel, o string) (op, inv *pyjson.Object, note string) {
	if s := relSlot(nodes, r); s != "" {
		return nil, nil, r.label + " on " + o + " is already there (slot " + s + "): no op"
	}
	op = obj("nm", "a", "h", h, "n", "r?", "b", r.relBody())
	if f := r.relFlags(); f != "" {
		op.Set("f", f)
	}
	return op, obj("nm", "v", "h", h, "n", "r?"), ""
}

// tagRelRemoveOp removes the relation slot holding r; the inverse re-adds it under its name with
// its flags.
func tagRelRemoveOp(h any, nodes *pyjson.Object, r tagRel, o string) (op, inv *pyjson.Object, err error) {
	s := relSlot(nodes, r)
	if s == "" {
		return nil, nil, plan.ToolError(o + " has no " + r.label + " (n4_read_tags lists its tag groups and relations)")
	}
	inv = obj("nm", "a", "h", h, "n", s, "b", r.relBody())
	if f, _ := get(box.Node(nodes, s), "f").(string); f != "" {
		inv.Set("f", f)
	}
	return obj("nm", "v", "h", h, "n", s), inv, nil
}
