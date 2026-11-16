package readtools

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// expectFilter is an expect_histories / expect_alarms argument: the points the caller expects
// to carry the extension. A nil filter keeps every point.
type expectFilter struct {
	kinds   []string // the texts of expectKinds, any of which the type name must hold
	pattern string   // a glob over the point's name or display name ("" keeps every name)
}

// expectArg reads the filter argument key (nil when absent). A key it does not know, a kind
// outside expectKinds or a malformed glob is refused: a filter that silently kept everything,
// or nothing, would misreport.
func expectArg(args *pyjson.Object, key string) (*expectFilter, error) {
	v, ok := args.Get(key)
	if !ok {
		return nil, nil
	}
	kinds := make([]string, 0, len(expectKinds))
	for k := range expectKinds {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	refuse := func(why string) (*expectFilter, error) {
		return nil, toolError(commBadExpect, key, strings.Join(kinds, ", "), why)
	}
	f, spec := &expectFilter{}, v.(*pyjson.Object) // an object: the schema checked
	for _, k := range spec.Keys() {
		value, _ := spec.Get(k)
		switch k {
		case "kinds":
			list, ok := value.([]any)
			if !ok {
				return refuse("kinds is not a list")
			}
			for _, item := range list {
				name, _ := item.(string)
				text, known := expectKinds[name]
				if !known {
					return refuse(fmt.Sprintf("unknown kind %s", pyjson.Repr(item)))
				}
				f.kinds = append(f.kinds, text)
			}
		case "name_pattern":
			pattern, ok := value.(string)
			if _, err := path.Match(pattern, ""); !ok || err != nil {
				return refuse(fmt.Sprintf("name_pattern %s is not a glob", pyjson.Repr(value)))
			}
			f.pattern = pattern
		default:
			return refuse(fmt.Sprintf("unknown key %s", pyjson.StrRepr(k)))
		}
	}
	return f, nil
}

// match reports whether the point row is one the filter expects.
func (f *expectFilter) match(row *pyjson.Object) bool {
	if f == nil {
		return true
	}
	kind := len(f.kinds) == 0
	_, simple, _ := strings.Cut(cellText(row, "Type"), ":")
	for _, text := range f.kinds {
		kind = kind || strings.Contains(simple, text)
	}
	if !kind || f.pattern == "" {
		return kind
	}
	for _, name := range []string{cellText(row, "Name"), cellText(row, displayColumn)} {
		if ok, _ := path.Match(f.pattern, name); ok {
			return true
		}
	}
	return false
}
