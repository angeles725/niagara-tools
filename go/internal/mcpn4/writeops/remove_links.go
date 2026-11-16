package writeops

import (
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
// write-scope prefix holding the parent, or link_scan_ord when given. Links found refuse the
// plan unless acknowledge_links is true; then outgoing_links_broken lists them and the inverse
// records each as an outgoing_link n4_rollback re-creates (bench retro 2026-11-15 delta 20).
func linkScan(c plan.Client, args *pyjson.Object, scope *ord.Scope, nodes *pyjson.Object, handles map[string]string,
	keys *pyjson.Object, notes, inverse []any) ([]any, []any, error) {
	root, narrowed := scopeRootOf(scope, str(args, "parent_ord")), false
	if v, ok := args.Get("link_scan_ord"); ok && v != nil {
		root, narrowed = pyjson.Str(v), true // in scope: a plain slot ORD
	}
	found, err := scanOutgoingWith(c, root, handles, true)
	if err != nil {
		return nil, nil, err
	}
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
