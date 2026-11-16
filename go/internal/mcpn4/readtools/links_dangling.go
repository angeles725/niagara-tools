package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// danglingScopeNote is the scope_note of n4_find_dangling_outputs.
const danglingScopeNote = "Components are reported up to depth levels below ord. " +
	"Links are seen when their target is at most one level below " +
	"depth; an out slot used only by a deeper target, or by a " +
	"target outside the subtree, is reported as dangling. A link " +
	"source given by handle or by absolute slot path is resolved; " +
	"while any link reading out has an unresolved source " +
	"(unresolved_links), every out not proven used is reported " +
	"as unknown, never as dangling."

// danglingOutputs is the dangling-output check over one scan: the components up to depth below
// ord whose out slot no seen link reads (dangling), the same while a link that may read out has
// an unresolved source (unknown, never dangling), and those unresolved links.
func danglingOutputs(ord string, nodes *pyjson.Object, links []link, depth int) (dangling, unknown, unresolved []any, err error) {
	used := map[string]bool{} // the (source_path, "out") pairs, by path
	dangling, unknown, unresolved = []any{}, []any{}, []any{}
	undecided := false
	for _, l := range links {
		switch slot := l.sourceSlot.(type) {
		case []any, *pyjson.Object: // an unhashable set member
			return nil, nil, nil, typeError
		case string:
			if p, ok := l.sourcePath.(string); ok && slot == "out" {
				used[p] = true
			}
		}
		if !l.resolved {
			unresolved = append(unresolved, obj("target_path", l.targetPath, "link_name", l.linkName,
				"source_ord", l.sourceOrd, "source_slot", l.sourceSlot))
			undecided = undecided || l.sourceSlot == nil || l.sourceSlot == "out"
		}
	}
	for _, p := range nodes.Keys() {
		kind, _ := box.Node(nodes, p).Get("t")
		if level(p) > depth || isLinkType(kind) {
			continue
		}
		out := "out" // "out" in children(nodes, p)
		if p != "" {
			out = p + "/out"
		}
		if _, has := nodes.Get(out); has && !used[absPath(ord, p)] {
			entry := obj("path", absPath(ord, p), "type", kind)
			if undecided {
				unknown = append(unknown, entry)
			} else {
				dangling = append(dangling, entry)
			}
		}
	}
	return dangling, unknown, unresolved, nil
}

// FindDanglingOutputs is n4_find_dangling_outputs: the components up to depth below ord whose
// out slot no seen link reads. While a link that may read out has an unresolved source, every
// out not proven used is unknown, never dangling.
func (t *Tools) FindDanglingOutputs(args *pyjson.Object) (*pyjson.Object, error) {
	ord, nodes, links, depth, err := t.scan(args)
	if err != nil {
		return nil, err
	}
	dangling, unknown, unresolved, err := danglingOutputs(ord, nodes, links, depth)
	if err != nil {
		return nil, err
	}
	if len(dangling) > 0 { // reads leave no audit line: remember it for the session retro
		t.Ctx.Observations = append(t.Ctx.Observations, obj("ts", session.Now(), "ord", ord, "count", len(dangling)))
	}
	return obj("ord", ord, "dangling", dangling, "unknown", unknown, "unresolved_links", unresolved,
		"scope_note", danglingScopeNote), nil
}
