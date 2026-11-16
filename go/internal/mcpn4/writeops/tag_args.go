package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// Bounds of one n4_set_tags batch.
const (
	MaxTagTargets = 50
	MaxTagOps     = 200
)

// msgAdHocInstalled cites the Tagging guide (Adding ad hoc tags): an ad hoc tag must not use the
// namespace of an installed dictionary, so allow_ad_hoc never permits it.
const msgAdHocInstalled = " is not defined by the installed dictionary %s of namespace %s; an ad hoc tag must not " +
	"use an installed dictionary's namespace (Tagging guide, Adding ad hoc tags: \"do not use a namespace that " +
	"is identical to an existing tag dictionary\"), even with allow_ad_hoc. Use a namespace of your own, " +
	"or n4_list_tag_dictionaries for the tags it defines"

// tagAdd is one requested tag: its id and the BOX body every target receives.
type tagAdd struct {
	id   tags.ID
	body *pyjson.Object
}

// tagOrdsArg is the scope list: the target ORDs (validated by the plan).
func tagOrdsArg(args *pyjson.Object) []any {
	v, _ := args.Get("ords")
	list, _ := v.([]any)
	return list
}

// tagTargets are the distinct target ORDs, 1 to MaxTagTargets.
func tagTargets(args *pyjson.Object) ([]string, error) {
	list := tagOrdsArg(args)
	if len(list) == 0 || len(list) > MaxTagTargets {
		return nil, plan.ToolError("ords must list 1 to 50 component ORDs")
	}
	seen, out := map[string]bool{}, []string{}
	for _, v := range list {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, plan.ToolError("ords must be component ORD strings")
		}
		if seen[s] {
			return nil, plan.ToolError("ords lists " + s + " twice")
		}
		seen[s], out = true, append(out, s)
	}
	return out, nil
}

// tagID parses one id, refusing anything but ns:name.
func tagID(v any) (tags.ID, error) {
	s, _ := v.(string)
	id, ok := tags.ParseID(s)
	if !ok {
		return tags.ID{}, plan.ToolError("a tag id must be ns:name (letters, digits, underscore), got " + pyjson.Repr(v))
	}
	return id, nil
}

// tagAdds resolves each add entry against the catalog: the definition gives the type and the
// default; an undefined tag needs allow_ad_hoc and a namespace no dictionary installs.
func tagAdds(args *pyjson.Object, cat *tags.Catalog) ([]tagAdd, error) {
	v, _ := args.Get("add")
	list, _ := v.([]any)
	adHoc, _ := args.Get("allow_ad_hoc")
	out, seen := []tagAdd{}, map[tags.ID]bool{}
	for _, e := range list {
		entry, ok := e.(*pyjson.Object)
		if !ok {
			return nil, plan.ToolError("each add entry is an object {id, value?, type?}")
		}
		idv, _ := entry.Get("id")
		id, err := tagID(idv)
		if err != nil {
			return nil, err
		}
		if seen[id] {
			return nil, plan.ToolError("add names " + id.String() + " twice")
		}
		seen[id] = true
		value, _ := entry.Get("value")
		typv, given := entry.Get("type")
		typ, _ := typv.(string)
		def, defined, err := cat.Def(id)
		if err != nil {
			return nil, err
		}
		var dflt any
		switch {
		case defined && def.ValueType == "":
			return nil, plan.ToolError(id.String() + " is a smart tag (" + def.Info + "): the station computes it; it is not added as a direct tag")
		case defined:
			if given && typ != def.ValueType {
				return nil, plan.ToolError(id.String() + " is defined as " + def.ValueType + ", not " + typ)
			}
			typ, dflt = def.ValueType, def.Default
		case cat.Installed(id.Namespace):
			d := cat.Dictionary(id.Namespace)
			return nil, plan.ToolError(id.String() + fmt.Sprintf(msgAdHocInstalled, d.Name, id.Namespace))
		case adHoc != true:
			return nil, plan.ToolError(id.String() + " is not defined by an installed tag dictionary; pass allow_ad_hoc: true " +
				"to add it as an ad hoc tag (n4_list_tag_dictionaries lists the defined tags)")
		case !given:
			typ = tags.InferType(value)
		}
		body, err := tags.Body(typ, value, dflt)
		if err != nil {
			return nil, plan.ToolError(id.String() + ": " + plan.PyStr(err))
		}
		out = append(out, tagAdd{id, body})
	}
	return out, nil
}

// tagRemoves are the ids of the remove list.
func tagRemoves(args *pyjson.Object) ([]tags.ID, error) {
	v, _ := args.Get("remove")
	list, _ := v.([]any)
	out := []tags.ID{}
	for _, e := range list {
		id, err := tagID(e)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}
