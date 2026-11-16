package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// dictEntry is the reply entry of one dictionary; budget counts the definitions still listed
// (tags, then groups, then relations) and cut says some were left out.
func dictEntry(d tags.Dictionary, defs []tags.TagDef, groups []tags.GroupDef, rels []string, budget *int) (*pyjson.Object, bool) {
	cut := false
	take := func() bool {
		if *budget <= 0 {
			cut = true
			return false
		}
		*budget--
		return true
	}
	tagList, groupList, relList := []any{}, []any{}, []any{}
	for _, def := range defs {
		if take() {
			var validity any
			if def.Validity != nil {
				validity, _ = def.Validity.Get("display")
			}
			tagList = append(tagList, obj("id", def.ID, "info", def.Info, "type", nilIfEmpty(def.ValueType),
				"default", def.Default, "validity", validity))
		}
	}
	for _, g := range groups {
		if take() {
			implied := []any{}
			for _, id := range g.Tags {
				implied = append(implied, id)
			}
			groupList = append(groupList, obj("id", g.ID, "group_ord", g.Ord, "tags", implied))
		}
	}
	for _, r := range rels {
		if take() {
			relList = append(relList, r)
		}
	}
	return obj("name", d.Name, "ord", d.Ord, "namespace", d.Namespace, "enabled", d.Enabled, "frozen", d.Frozen,
		"version", d.Version, "status", d.Status, "tags", tagList, "tag_groups", groupList, "relations", relList), cut
}
