package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// LinkScanDepth is LINK_SCAN_DEPTH: how deep the optional outgoing-link scan loads.
const LinkScanDepth = 6

// scanNotes is the end of _remove_plan: with link_scan_ord, the links from the subtree into
// components outside it (keys gains outgoing_links_broken); without, a note when the subtree
// has outputs such links could read.
func scanNotes(c plan.Client, args *pyjson.Object, nodes *pyjson.Object, handles map[string]string,
	keys *pyjson.Object, notes []any) ([]any, error) {
	scanArg, ok := args.Get("link_scan_ord")
	if !ok || scanArg == nil {
		if hasOutputs(box.Node(nodes, "")) {
			notes = append(notes, "outgoing links to components outside the removed subtree are stored on "+
				"those targets: they cannot be detected from the subtree load and will "+
				"be broken by the remove; pass "+
				"link_scan_ord (inside the write scope) to scan a wider root")
		}
		return notes, nil
	}
	scanOrd := pyjson.Str(scanArg) // in scope: a plain slot ORD
	found, err := scanOutgoing(c, scanOrd, handles)
	if err != nil {
		return nil, err
	}
	head := "scanned " + scanOrd + " to depth " + strconv.Itoa(LinkScanDepth)
	if len(found) == 0 {
		return append(notes, head+": no outgoing links from this subtree found "+
			"(components deeper than that were not scanned)"), nil
	}
	keys.Set("outgoing_links_broken", found)
	return append(notes, head+": the remove will break "+strconv.Itoa(len(found))+" link(s) from this "+
		"subtree into components outside it (see outgoing_links_broken); they are not restored"), nil
}

// hasOutputs is _has_outputs: some descendant slot is a Status value, something links read.
func hasOutputs(node *pyjson.Object) bool {
	for _, c := range kidsOf(node, 1) {
		if statusType(get(c, "t")) || hasOutputs(c) {
			return true
		}
	}
	return false
}

// outgoing is _outgoing_links: links stored on a component outside the removed subtree (its
// handle not in handles) whose source is inside it.
func outgoing(node *pyjson.Object, path string, handles map[string]string, found []*pyjson.Object) ([]*pyjson.Object, error) {
	here := get(node, "h")
	for _, child := range kidsOf(node, 1) {
		if !isLink(get(child, "t")) {
			sub := name(child)
			if path != "" {
				sub = path + "/" + sub
			}
			var err error
			if found, err = outgoing(child, sub, handles, found); err != nil {
				return nil, err
			}
			continue
		}
		slots := pyjson.NewObject()
		for _, k := range kidsOf(child, 1) {
			slots.Set(name(k), get(k, "v"))
		}
		tail, err := sourceTail(slots)
		if err != nil {
			return nil, err
		}
		if !strings.HasPrefix(tail, "h:") {
			continue
		}
		source, in := handles["s"+tail[2:]]
		if !in {
			continue
		}
		key, err := pyjson.DictKey(here)
		if err != nil {
			return nil, unhashable(here)
		}
		if _, inside := handles[key]; !inside {
			found = append(found, obj("path", path, "target_slot", get(slots, "targetSlotName"),
				"source_path", source, "source_slot", get(slots, "sourceSlotName")))
		}
	}
	return found, nil
}

// scanOutgoing is _scan_outgoing: the outgoing links under scanOrd, sorted by target, target
// slot and source path.
func scanOutgoing(c plan.Client, scanOrd string, handles map[string]string) ([]any, error) {
	nodes, err := c.LoadTree(scanOrd, LinkScanDepth, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	found, err := outgoing(box.Node(nodes, ""), "", handles, nil)
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(found))
	for _, f := range found {
		target, err := ord.JoinOrd(scanOrd, get(f, "path").(string))
		if err != nil {
			return nil, err
		}
		out = append(out, obj("target", target, "target_slot", get(f, "target_slot"),
			"source_path", get(f, "source_path"), "source_slot", get(f, "source_slot")))
	}
	return out, pySort(out, func(e any) []any {
		o := e.(*pyjson.Object)
		return []any{get(o, "target"), get(o, "target_slot"), get(o, "source_path")}
	})
}
