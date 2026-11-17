package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// copySource is one source component as n4_copy_component and n4_replicate_points capture it:
// the loaded tree, its type contracts, the configured body every copy adds, the nodes left out
// and the links found inside it (never copied).
type copySource struct {
	name     string
	root     *pyjson.Object
	ct       *plan.Contracts
	body     *pyjson.Object
	stripped []strippedNode
	links    int
}

// loadCopySource loads child source of parentOrd one level deeper than CopyDepth (anything there
// means a copy would be partial, refused) and strips it to its configured body. parentNodes is
// parentOrd loaded one level deep, where source must exist.
func loadCopySource(c plan.Client, parentOrd, source string, parentNodes *pyjson.Object) (*copySource, error) {
	if box.Node(parentNodes, source) == nil {
		return nil, plan.ToolError("component " + pyjson.StrRepr(source) + " not found under " + parentOrd)
	}
	childOrd, err := ord.ChildOrd(parentOrd, source)
	if err != nil {
		return nil, err
	}
	_, nodes, err := plan.Handle(c, childOrd, CopyDepth+1)
	if err != nil {
		return nil, err
	}
	root := box.Node(nodes, "")
	if treeDepth(root) > CopyDepth {
		return nil, plan.ToolError(pyjson.StrRepr(source) + " is deeper than the copy depth (" +
			strconv.Itoa(CopyDepth) + " levels): a copy would miss part of it; copy a smaller component")
	}
	ct, err := copyContracts(c, root)
	if err != nil {
		return nil, err
	}
	body, stripped, s, err := configured(ct, root)
	if err != nil {
		return nil, err
	}
	return &copySource{source, root, ct, body, stripped, len(s.links)}, nil
}

// linksNote is the note of a source holding links a copy does not carry; nil when it has none.
func (s *copySource) linksNote() []any {
	if s.links == 0 {
		return nil
	}
	return []any{strconv.Itoa(s.links) + " link(s) inside " + pyjson.StrRepr(s.name) +
		" are not copied: create them on each copy with n4_create_link"}
}
