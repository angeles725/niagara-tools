package px

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyxml"
)

// Binding is one ord attribute of a binding element (a tag ending in Binding: ValueBinding,
// ActionBinding, a kitPx or vendor binding) in a Px view: the element, the name of the nearest
// enclosing widget that has one ("" when none) and the ORD as written, spaces trimmed.
type Binding struct {
	Element string
	Widget  string
	Ord     string
}

// Bindings is every binding ORD of a Px document in document order, read from its <content>
// like Parse; the document is refused on the same grounds as Parse (a ValueError).
func Bindings(text string) ([]Binding, error) {
	root, err := parseRoot(text)
	if err != nil {
		return nil, err
	}
	out := []Binding{}
	var walk func(el *pyxml.Element, widget string)
	walk = func(el *pyxml.Element, widget string) {
		if strings.HasSuffix(el.Tag, "Binding") {
			if ord, _ := el.Get("ord"); strip(ord) != "" {
				out = append(out, Binding{Element: el.Tag, Widget: widget, Ord: strip(ord)})
			}
		} else if name, _ := el.Get("name"); name != "" {
			widget = name
		}
		for _, c := range el.Children {
			walk(c, widget)
		}
	}
	walk(contentOf(root), "")
	return out, nil
}
