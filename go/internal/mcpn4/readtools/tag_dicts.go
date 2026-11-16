package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// Bounds of n4_list_tag_dictionaries: definitions listed by default and at most, across all.
const (
	TagDictDefaultItems = 200
	TagDictMaxItems     = 2000
)

// ListTagDictionaries is n4_list_tag_dictionaries: the installed dictionaries (one BQL query),
// each with its tag definitions, tag groups and relation ids (three loads per dictionary listed).
func (t *Tools) ListTagDictionaries(args *pyjson.Object) (*pyjson.Object, error) {
	budget := intArg(args, "max_items", TagDictDefaultItems) // 1..TagDictMaxItems: the registry schema refuses others
	dicts, err := t.dictionaries()
	if err != nil {
		return nil, err
	}
	if want, ok := args.Get("dictionary"); ok && want != nil {
		if dicts, err = pickDictionary(dicts, pyjson.Str(want)); err != nil {
			return nil, err
		}
	}
	client := t.Ctx.Session.Client.(Client)
	out, truncated := []any{}, false
	for _, d := range dicts {
		defs, err := tags.TagDefs(client, d)
		if err != nil {
			return nil, err
		}
		groups, err := tags.GroupDefs(client, d)
		if err != nil {
			return nil, err
		}
		rels, err := tags.RelationDefs(client, d)
		if err != nil {
			return nil, err
		}
		entry, cut := dictEntry(d, defs, groups, rels, &budget)
		out, truncated = append(out, entry), truncated || cut
	}
	t.recordRead("n4_list_tag_dictionaries", "dictionary", args, len(out))
	return obj("dictionaries", out, "truncated", truncated), nil
}

// dictionaries are the installed dictionaries, from tags.DictionaryQuery.
func (t *Tools) dictionaries() ([]tags.Dictionary, error) {
	columns, rows, _, _, err := t.query(strings.TrimSuffix(tags.ServiceOrd, "/"), tags.DictionaryQuery, 500, 60)
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, tags.DictionaryColumns)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	out := make([]tags.Dictionary, 0, len(keyed))
	for _, r := range keyed {
		out = append(out, tags.FromRow(r))
	}
	return out, nil
}

// pickDictionary keeps the dictionary named want or whose namespace is want.
func pickDictionary(dicts []tags.Dictionary, want string) ([]tags.Dictionary, error) {
	names := []string{}
	for _, d := range dicts {
		if d.Name == want || d.Namespace == want {
			return []tags.Dictionary{d}, nil
		}
		names = append(names, d.Name+" ("+d.Namespace+")")
	}
	return nil, toolError("no tag dictionary named or with namespace %s; installed: %s", pyjson.StrRepr(want),
		strings.Join(names, ", "))
}
