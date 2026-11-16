package tags

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
)

// Catalog is what the installed dictionaries define, read for a write: the dictionaries by
// namespace (one load of the service) and, for each namespace a write names, its tag
// definitions (one load each, on first use).
type Catalog struct {
	l      Loader
	dicts  map[string]Dictionary
	defs   map[string]map[string]TagDef
	groups map[string]map[string]GroupDef
	rels   map[string]map[string]bool
}

// LoadCatalog reads the dictionaries under the service. A dictionary whose load omits its
// namespace (the Niagara dictionary: the slot is at its declared default) takes the namespace its
// type declares, read from the type registry when l can read it.
func LoadCatalog(l Loader) (*Catalog, error) {
	nodes, err := l.LoadTree(ServiceOrd, 2, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	c := &Catalog{l: l, dicts: map[string]Dictionary{}, defs: map[string]map[string]TagDef{},
		groups: map[string]map[string]GroupDef{}, rels: map[string]map[string]bool{}}
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		typ, _ := get(box.Node(kids, name), "t").(string)
		if !strings.HasSuffix(typ, "TagDictionary") {
			continue
		}
		ns, _ := get(box.Node(nodes, name+"/namespace"), "v").(string)
		if tl, ok := l.(plan.TypeLoader); ns == "" && ok {
			if def, err := plan.DeclaredDefault(tl, typ, "namespace"); err == nil && def != nil {
				ns, _ = get(def, "v").(string)
			}
		}
		if ns != "" {
			c.dicts[ns] = Dictionary{Name: name, Ord: ServiceOrd + "/" + name, Namespace: ns}
		}
	}
	return c, nil
}

// Installed is whether a dictionary installs namespace ns.
func (c *Catalog) Installed(ns string) bool { _, ok := c.dicts[ns]; return ok }

// Def is the definition of id in its dictionary, ok false when the namespace is not installed or
// does not define the name.
func (c *Catalog) Def(id ID) (TagDef, bool, error) {
	d, ok := c.dicts[id.Namespace]
	if !ok {
		return TagDef{}, false, nil
	}
	byName, read := c.defs[id.Namespace]
	if !read {
		list, err := TagDefs(c.l, d)
		if err != nil {
			return TagDef{}, false, err
		}
		byName = map[string]TagDef{}
		for _, def := range list {
			byName[def.Name] = def
		}
		c.defs[id.Namespace] = byName
	}
	def, ok := byName[id.Name]
	return def, ok, nil
}

// Dictionary is the installed dictionary of namespace ns (Name "" when none).
func (c *Catalog) Dictionary(ns string) Dictionary { return c.dicts[ns] }

// Group is the tag group definition of id, ok false when its dictionary does not define it.
func (c *Catalog) Group(id ID) (GroupDef, bool, error) {
	d, ok := c.dicts[id.Namespace]
	if !ok {
		return GroupDef{}, false, nil
	}
	byName, read := c.groups[id.Namespace]
	if !read {
		list, err := GroupDefs(c.l, d)
		if err != nil {
			return GroupDef{}, false, err
		}
		byName = map[string]GroupDef{}
		for _, g := range list {
			byName[g.Name] = g
		}
		c.groups[id.Namespace] = byName
	}
	g, ok := byName[id.Name]
	return g, ok, nil
}

// RelationDefined is whether the dictionary of id's namespace defines the relation id.
func (c *Catalog) RelationDefined(id ID) (bool, error) {
	d, ok := c.dicts[id.Namespace]
	if !ok {
		return false, nil
	}
	set, read := c.rels[id.Namespace]
	if !read {
		list, err := RelationDefs(c.l, d)
		if err != nil {
			return false, err
		}
		set = map[string]bool{}
		for _, r := range list {
			set[r] = true
		}
		c.rels[id.Namespace] = set
	}
	return set[id.String()], nil
}
