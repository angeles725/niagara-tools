package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Source ORDs name a component by handle, "h:<handle>", or by slot path, "slot:/Folder/A";
// either may be prefixed by the station as "station:|...". Only the part after the last "|" is
// used.
const (
	handlePrefix = "h:"
	slotPrefix   = "slot:"
)

// level is _level: how many slot names deep path is.
func level(path string) int {
	if path == "" {
		return 0
	}
	return strings.Count(path, "/") + 1
}

// absPath is _abs_path: the absolute slot path of path below ord.
func absPath(ord, path string) string {
	base := "/"
	if _, after, ok := strings.Cut(ord, "slot:"); ok {
		base = after
	}
	base = strings.TrimRight(base, "/")
	switch {
	case path != "":
		return base + "/" + path
	case base == "":
		return "/"
	}
	return base
}

// sourcePath is _source_path: the path of a link's source component (nil when unknown) and
// whether the source is resolved. A handle is looked up among the loaded components (nil:
// outside the subtree); an absolute slot path is the path itself; any other form is not
// resolved. A truthy source ORD that is not a str is Python's AttributeError.
func sourcePath(byHandle map[string]string, sourceOrd any) (any, bool, error) {
	s, ok := sourceOrd.(string)
	if !ok && pyjson.Truthy(sourceOrd) {
		return nil, false, attributeError
	}
	last := s[strings.LastIndex(s, "|")+1:]
	if h, ok := strings.CutPrefix(last, handlePrefix); ok {
		if p, ok := byHandle[h]; ok {
			return p, true, nil
		}
		return nil, true, nil
	}
	if p, ok := strings.CutPrefix(last, slotPrefix); ok && strings.HasPrefix(p, "/") {
		if p = strings.TrimRight(p, "/"); p == "" {
			p = "/"
		}
		return p, true, nil
	}
	return nil, false, nil
}

// link is one link _scan found; targetLevel and resolved are for its callers only.
type link struct {
	targetPath, linkName   string
	sourceOrd, sourcePath  any
	sourceSlot, targetSlot any
	targetLevel            int
	resolved               bool
}

// scan is _scan: scanOrd over the ord and depth arguments.
func (t *Tools) scan(args *pyjson.Object) (string, *pyjson.Object, []link, int, error) {
	ord, err := ordArg(args)
	if err != nil {
		return "", nil, nil, 0, err
	}
	nodes, links, err := t.scanOrd(ord, intArg(args, "depth", 2))
	return ord, nodes, links, intArg(args, "depth", 2), err
}

// scanOrd is ord loaded depth+3 deep (a link sits one level below its target and its slots
// one more) and every link whose target is at most depth+1 levels below ord, so that an out
// slot feeding a target one level below depth is seen as used.
func (t *Tools) scanOrd(ord string, depth int) (*pyjson.Object, []link, error) {
	nodes, err := t.Ctx.Session.Client.(Client).LoadTree(ord, depth+3, box.LoadOptions{})
	if err != nil {
		return nil, nil, err
	}
	byHandle := map[string]string{}
	for _, p := range nodes.Keys() {
		switch h, _ := box.Node(nodes, p).Get("h"); x := h.(type) {
		case []any, *pyjson.Object: // an unhashable dict key
			if pyjson.Truthy(x) {
				return nil, nil, typeError
			}
		case string:
			if x != "" {
				byHandle[x] = absPath(ord, p)
			}
		} // any other handle is never a "h:" lookup key
	}
	var links []link
	for _, p := range nodes.Keys() {
		if kind, _ := box.Node(nodes, p).Get("t"); !isLinkType(kind) || level(p)-1 > depth+1 {
			continue
		}
		kids := box.Children(nodes, p)
		sourceOrd := box.ChildValue(kids, "sourceOrd", nil)
		source, resolved, err := sourcePath(byHandle, sourceOrd)
		if err != nil {
			return nil, nil, err
		}
		target, name := "", p
		if i := strings.LastIndex(p, "/"); i >= 0 {
			target, name = p[:i], p[i+1:]
		}
		links = append(links, link{targetPath: absPath(ord, target), linkName: name, sourceOrd: sourceOrd,
			sourcePath: source, sourceSlot: box.ChildValue(kids, "sourceSlotName", nil),
			targetSlot: box.ChildValue(kids, "targetSlotName", nil), targetLevel: level(target), resolved: resolved})
	}
	return nodes, links, nil
}

// ListLinks is n4_list_links: the links whose target component is at most depth levels below ord.
func (t *Tools) ListLinks(args *pyjson.Object) (*pyjson.Object, error) {
	ord, _, links, depth, err := t.scan(args)
	if err != nil {
		return nil, err
	}
	shown := []any{}
	for _, l := range links {
		if l.targetLevel <= depth {
			shown = append(shown, obj("target_path", l.targetPath, "link_name", l.linkName,
				"source_ord", l.sourceOrd, "source_path", l.sourcePath, "source_slot", l.sourceSlot,
				"target_slot", l.targetSlot))
		}
	}
	return obj("ord", ord, "links", shown), nil
}
