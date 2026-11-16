package tags

import "github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"

// RelationType is the type of a relation slot (javax.baja.sys.BRelation, a struct).
const RelationType = "baja:Relation"

// TagGroupRelation is the relation id that applies a tag group: an outbound relation from the
// component to the group's definition in a dictionary (BNiagaraTagDictionary.TAG_GROUP_RELATION).
const TagGroupRelation = "n:tagGroup"

// Relation is one direct relation slot as a load shows it: its id, the endpoint ORD (sourceOrd)
// and whether it is inbound.
type Relation struct {
	Slot, ID, Endpoint string
	Inbound            bool
}

// RelationFields is a relation from the values of its struct children (relationId, sourceOrd,
// inbound); a field the load omits is at its default ("", "", false).
func RelationFields(slot string, fields map[string]any) Relation {
	r := Relation{Slot: slot}
	r.ID, _ = fields["relationId"].(string)
	r.Endpoint, _ = fields["sourceOrd"].(string)
	r.Inbound = fields["inbound"] == "true"
	return r
}

// IsTagGroup is whether the relation applies a tag group.
func (r Relation) IsTagGroup() bool { return r.ID == TagGroupRelation && !r.Inbound }

// RelationBody is the BOX body of an outbound relation id to the endpoint sourceOrd (measured on
// 4.15: relationId a baja:String, sourceOrd a baja:Ord; the station fills relationTags).
func RelationBody(id, sourceOrd string) *pyjson.Object {
	field := func(n, t, v string) *pyjson.Object {
		o := pyjson.NewObject()
		o.Set("nm", "p")
		o.Set("n", n)
		o.Set("t", t)
		o.Set("v", v)
		return o
	}
	body := pyjson.NewObject()
	body.Set("nm", "p")
	body.Set("t", RelationType)
	body.Set("s", []any{field("relationId", "baja:String", id), field("sourceOrd", "baja:Ord", sourceOrd)})
	return body
}
