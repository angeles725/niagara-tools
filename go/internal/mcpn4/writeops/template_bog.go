package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// tplMaxNodes bounds the property nodes of one template tree (one add op carries them all).
const tplMaxNodes = 5000

// tplNode is one <p> of template.bog: its slot name (escaped, as the station stores it), handle,
// type spec with the module alias resolved (bog "b:Folder" is "baja:Folder"), value, instance
// flags, whether it carries slot facets (x) or categories (c), and its child properties.
// Actions (<a>) are not kept: a deploy, like a copy, does not carry them.
type tplNode struct {
	name, handle, typ, value, flags string
	hasValue, facets, categories    bool
	kids                            []*tplNode
}

// tplBog is template.bog read: the template component (the first child of the bog's root
// folder, template-rt BNtplFile.getBaseComponent) and how many root children the bog holds.
type tplBog struct {
	base      *tplNode
	rootKids  int
	nodeCount int
}

// parseTemplateBog reads the bog's <p> tree; module aliases (m="b=baja c=control") hold for the
// rest of the document once declared, as the bog encoder writes them.
func parseTemplateBog(data []byte) (*tplBog, error) {
	doc, err := xmlsrc.Parse(tplBogEntry, data)
	if err != nil {
		return nil, plan.ToolError("template.bog is not well-formed XML: " + err.Error())
	}
	elems := doc.Elements()
	if len(elems) == 0 || elems[0].Name != "bajaObjectGraph" {
		return nil, plan.ToolError("template.bog has no <bajaObjectGraph> root")
	}
	aliases := map[string]string{}
	nodes := make([]*tplNode, len(elems))
	b := &tplBog{}
	for i, e := range elems {
		if e.Name != "p" || e.Parent < 0 {
			continue
		}
		parent := nodes[e.Parent]
		if parent == nil && e.Parent != 0 {
			continue // a property under an action or unknown element
		}
		if m, ok := e.Attr("m"); ok {
			for _, decl := range strings.Fields(m) {
				if alias, module, found := strings.Cut(decl, "="); found {
					aliases[alias] = module
				}
			}
		}
		n, err := bogNode(e, aliases)
		if err != nil {
			return nil, err
		}
		if b.nodeCount++; b.nodeCount > tplMaxNodes {
			return nil, plan.ToolError("template.bog holds more than " + strconv.Itoa(tplMaxNodes) +
				" properties: too large for one deploy")
		}
		nodes[i] = n
		if parent != nil {
			parent.kids = append(parent.kids, n)
		}
	}
	root := firstNode(doc, nodes, 0)
	if root == nil {
		return nil, plan.ToolError("template.bog holds no component")
	}
	b.rootKids = len(root.kids)
	if b.rootKids == 0 {
		return nil, plan.ToolError("template.bog's root folder holds no template component")
	}
	b.base = root.kids[0]
	return b, nil
}

// firstNode is the first <p> child of element i.
func firstNode(doc *xmlsrc.Doc, nodes []*tplNode, i int) *tplNode {
	for _, j := range doc.Children(i) {
		if nodes[j] != nil {
			return nodes[j]
		}
	}
	return nil
}

func bogNode(e xmlsrc.Element, aliases map[string]string) (*tplNode, error) {
	n := &tplNode{}
	n.name, _ = e.Attr("n")
	n.handle, _ = e.Attr("h")
	n.flags, _ = e.Attr("f")
	n.value, n.hasValue = e.Attr("v")
	_, n.facets = e.Attr("x")
	_, n.categories = e.Attr("c")
	if t, ok := e.Attr("t"); ok {
		alias, typ, found := strings.Cut(t, ":")
		module, known := aliases[alias]
		if !found || !known {
			return nil, plan.ToolError("template.bog: type " + t + " of " + n.name +
				" uses a module alias the bog does not declare")
		}
		n.typ = module + ":" + typ
	}
	return n, nil
}

// child is n's child named name; nil when absent.
func (n *tplNode) child(name string) *tplNode {
	for _, k := range n.kids {
		if k.name == name {
			return k
		}
	}
	return nil
}
