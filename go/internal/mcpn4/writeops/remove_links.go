package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// removePlanWith is the remove plan the server runs (bench retro 2026-11-15 delta 3): the plain
// plan (the Python port, which scans only with link_scan_ord) with the link scan always on.
func removePlanWith(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	return removePlanIn(c, args, w.Scope)
}

// linkSlot reports whether the node at path p is a slot of a link (its parent is one).
func linkSlot(nodes *pyjson.Object, p string) bool {
	head, _ := rpartition(p)
	return p != "" && head != "" && isLink(get(box.Node(nodes, head), "t"))
}

// linkScan scans root for links from the subtree (handles) into components outside it: the
// write-scope prefix holding the parent, or link_scan_ord when given (equal to or under that
// prefix: it only narrows). Links found refuse the plan unless acknowledge_links is true; then
// outgoing_links_broken lists them and the inverse records each as an outgoing_link n4_rollback
// re-creates (bench retro 2026-11-15 delta 20).
func linkScan(c plan.Client, args *pyjson.Object, scope *ord.Scope, nodes *pyjson.Object, handles map[string]string,
	slot string, keys *pyjson.Object, notes, inverse []any) ([]any, []any, error) {
	parent := str(args, "parent_ord")
	root, narrowed := scopeRootOf(scope, parent), false
	if v, ok := args.Get("link_scan_ord"); ok && v != nil {
		scan := pyjson.Str(v) // in scope: a plain slot ORD
		if !ordUnder(scan, root) {
			return nil, nil, plan.ToolError(scanOrdOutside(scan, root))
		}
		root, narrowed = scan, true
	}
	removed, err := ord.ChildOrd(parent, str(args, "name"))
	if err != nil {
		return nil, nil, err
	}
	found, err := scanOutgoingFrom(c, root, removed, handles, narrowed)
	if err != nil {
		return nil, nil, err
	}
	found = fromSlot(found, slot) // a removed plain slot breaks only the links it sources
	head := scanHead(root, narrowed)
	if len(found) == 0 { // nothing seen: a subtree with outputs gets the stronger warning
		return append(notes, head+scanNone+scanBlind(root, hasOutputs(box.Node(nodes, "")))), inverse, nil
	}
	if ack, _ := args.Get("acknowledge_links"); ack != true {
		return nil, nil, plan.ToolError(linksRefusal(head, root, found))
	}
	shown := make([]any, len(found))
	for i, f := range found { // the target's handle is for the rollback, not the plan's list
		o := f.(*pyjson.Object)
		shown[i] = obj("target", get(o, "target"), "target_slot", get(o, "target_slot"),
			"source_path", get(o, "source_path"), "source_slot", get(o, "source_slot"))
		inverse = append(inverse, obj("outgoing_link", o))
	}
	keys.Set("outgoing_links_broken", shown)
	return append(notes, head+scanBroken(len(found))+scanBlind(root, false)), inverse, nil
}

// scanOutgoingFrom loads root to LinkScanDepth and returns its outgoing links with each target's
// handle. A root holding more than LinkScanNodeBudget nodes refuses (never a silent skip), and a
// link whose target is the removed component or under it is internal whatever its depth: the
// snapshot's handles reach only depth 3 (native review R3-deep-internal-links-flagged).
func scanOutgoingFrom(c plan.Client, root, removed string, handles map[string]string, narrowed bool) ([]any, error) {
	nodes, err := c.LoadTree(root, LinkScanDepth, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	if n := len(nodes.Keys()); n > LinkScanNodeBudget {
		return nil, plan.ToolError(scanTooLarge(root, n, narrowed))
	}
	all, err := matchOutgoing(nodes, root, handles, true)
	if err != nil {
		return nil, err
	}
	found := make([]any, 0, len(all))
	for _, f := range all {
		if !ordUnder(pyjson.Str(get(f.(*pyjson.Object), "target")), removed) {
			found = append(found, f)
		}
	}
	return found, nil
}

// ordUnder reports whether slot ORD o is root or under it, on slot boundaries.
func ordUnder(o, root string) bool {
	o, root = strings.TrimRight(o, "/"), strings.TrimRight(root, "/")
	return o == root || strings.HasPrefix(o, root+"/")
}
