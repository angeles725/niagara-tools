package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// impliedTags is the impliedTags call on the component with handle h: each tag as {id, type,
// value}, in the station's order; a key without a known type letter keeps its raw key as id and
// a null type (reported, never guessed).
func impliedTags(c sessionCaller, h any) ([]any, error) {
	r, err := c.SSC("impliedTags", h)
	if err != nil {
		return nil, err
	}
	m, ok := r.(*pyjson.Object)
	if !ok {
		return nil, toolError("impliedTags answered %s, not an object", pyjson.Repr(r))
	}
	out := []any{}
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		if tag, ok := tags.DecodeImplied(k, v); ok {
			out = append(out, obj("id", tag.ID, "type", tag.Type, "value", tag.Value))
		} else {
			out = append(out, obj("id", k, "type", nil, "value", v))
		}
	}
	return out, nil
}

// impliedRelations is the impliedRelations call on the component with handle h: each relation
// as {id, endpoint_ord, inbound}.
func impliedRelations(c sessionCaller, h any) ([]any, error) {
	r, err := c.SSC("impliedRelations", h)
	if err != nil {
		return nil, err
	}
	list, ok := r.([]any)
	if !ok {
		return nil, toolError("impliedRelations answered %s, not a list", pyjson.Repr(r))
	}
	out := []any{}
	for _, raw := range list {
		rel, ok := raw.(*pyjson.Object)
		if !ok {
			return nil, toolError("impliedRelations answered an entry %s, not an object", pyjson.Repr(raw))
		}
		out = append(out, obj("id", lookup(rel, "rId"), "endpoint_ord", lookup(rel, "entityOrd"),
			"inbound", lookup(rel, "inbound")))
	}
	return out, nil
}
