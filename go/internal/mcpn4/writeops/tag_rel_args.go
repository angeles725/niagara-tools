package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// tagGroups resolves add_groups or remove_groups: each id must name a tag group an installed
// dictionary defines (there are no ad hoc groups).
func tagGroups(args *pyjson.Object, key string, cat *tags.Catalog) ([]tagRel, error) {
	out := []tagRel{}
	for _, e := range asList(lookupArg(args, key)) {
		id, err := tagID(e)
		if err != nil {
			return nil, err
		}
		g, ok, err := cat.Group(id)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, plan.ToolError("no installed dictionary defines the tag group " + id.String() +
				" (n4_list_tag_dictionaries lists the groups)")
		}
		out = append(out, tagRel{label: "tag group " + id.String(), id: tags.TagGroupRelation, source: g.Ord,
			match: []string{g.Ord, "station:|" + g.Ord}, group: true, implies: g.Tags})
	}
	return out, nil
}

// tagRelations resolves add_relations or remove_relations: {id, endpoint_ord}; the endpoint is
// written as its handle ORD (h:..., as Workbench writes it) and matched in that form or as its
// slot ORD. An id no dictionary defines needs allow_ad_hoc and a namespace of its own.
func tagRelations(c plan.Client, args *pyjson.Object, key string, cat *tags.Catalog) ([]tagRel, error) {
	out := []tagRel{}
	adHoc := lookupArg(args, "allow_ad_hoc") == true
	for _, e := range asList(lookupArg(args, key)) {
		entry, ok := e.(*pyjson.Object)
		if !ok {
			return nil, plan.ToolError("each " + key + " entry is an object {id, endpoint_ord}")
		}
		id, err := tagID(get(entry, "id"))
		if err != nil {
			return nil, err
		}
		if id.String() == tags.TagGroupRelation {
			return nil, plan.ToolError("n:tagGroup relations are written with add_groups / remove_groups")
		}
		defined, err := cat.RelationDefined(id)
		if err != nil {
			return nil, err
		}
		switch {
		case defined:
		case cat.Installed(id.Namespace):
			return nil, plan.ToolError("relation " + id.String() + " is not defined by the installed dictionary " +
				cat.Dictionary(id.Namespace).Name + "; an ad hoc relation must use a namespace of its own")
		case !adHoc:
			return nil, plan.ToolError("relation " + id.String() + " is not defined by an installed tag dictionary; " +
				"pass allow_ad_hoc: true to add it as an ad hoc relation")
		}
		end, _ := get(entry, "endpoint_ord").(string)
		h, _, err := plan.Handle(c, end, 0)
		if err != nil {
			return nil, err
		}
		src := "h:" + pyjson.Str(h)
		out = append(out, tagRel{label: "relation " + id.String() + " to " + end, id: id.String(), source: src,
			match: []string{src, end, tags.SlotOrd(end)}})
	}
	return out, nil
}

func lookupArg(args *pyjson.Object, k string) any { v, _ := args.Get(k); return v }
