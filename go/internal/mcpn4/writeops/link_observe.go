package writeops

// The link read side: the recover hook after an ambiguous checkLinks reply, the read-back, and the
// slot helpers both share.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// linkSlots is _link_slots: a link's name and its three slot values.
func linkSlots(nodes *pyjson.Object, name string) *pyjson.Object {
	kids := box.Children(nodes, name)
	return obj("link", name, "sourceOrd", get(box.Node(kids, "sourceOrd"), "v"),
		"sourceSlotName", get(box.Node(kids, "sourceSlotName"), "v"),
		"targetSlotName", get(box.Node(kids, "targetSlotName"), "v"))
}

// sourceTail is `(d["sourceOrd"] or "").rsplit("|", 1)[-1]`: the station may prefix the handle
// ORD, so only `h:<handle>` is compared.
func sourceTail(d *pyjson.Object) (string, error) {
	switch s := get(d, "sourceOrd").(type) {
	case string:
		return s[strings.LastIndex(s, "|")+1:], nil
	default:
		if !pyjson.Truthy(s) {
			return "", nil
		}
		return "", &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(s) + "' object has no attribute 'rsplit'"}
	}
}

// handleOrd is "h:" + handle.
func handleOrd(h any) (string, error) {
	s, ok := h.(string)
	if !ok {
		return "", &plan.PyError{Type: "TypeError", Msg: `can only concatenate str (not "` + plan.TypeName(h) + `") to str`}
	}
	return "h:" + s, nil
}

// linkRecover is _link_recover: after an ambiguous reply, the one new link whose source and
// target slots are the requested ones, else (nil, []).
func linkRecover(c plan.Client, args *pyjson.Object, p *plan.Planned, _ []any) (any, []any, error) {
	d := p.Data.(linkData)
	_, nodes, err := plan.Handle(c, str(args, "target_ord"), 2)
	if err != nil {
		return nil, nil, err
	}
	var found []*pyjson.Object
	for _, name := range links(nodes) {
		if d.existing[name] {
			continue
		}
		f := linkSlots(nodes, name)
		tail, err := sourceTail(f)
		if err != nil {
			return nil, nil, err
		}
		want, err := handleOrd(d.srcH)
		if err != nil {
			return nil, nil, err
		}
		if tail == want && box.PyEqual(get(f, "sourceSlotName"), str(args, "source_slot")) &&
			box.PyEqual(get(f, "targetSlotName"), str(args, "target_slot")) {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		return nil, []any{}, nil
	}
	return found[0], []any{obj("nm", "v", "h", d.tgtH, "n", get(found[0], "link"))}, nil
}

func linkReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	result, err := linkResult(replies)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if len(inverse) == 0 { // refused by the station (see linkInverse)
		return nil, result, nil, "failed", nil
	}
	name := get(inverse[0].(*pyjson.Object), "n")
	src, err := handleOrd(p.Data.(linkData).srcH)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested := obj("link", name, "sourceOrd", src, "sourceSlotName", str(args, "source_slot"),
		"targetSlotName", str(args, "target_slot"))
	_, nodes, err := plan.Handle(c, str(args, "target_ord"), 2)
	if err != nil {
		return nil, nil, nil, "", err
	}
	var observed *pyjson.Object
	switch n := name.(type) {
	case []any, *pyjson.Object: // `name in nodes` hashes it
		return nil, nil, nil, "", unhashable(name)
	case string:
		if box.Node(nodes, n) != nil {
			observed = linkSlots(nodes, n)
		}
	}
	want, err := tail(requested)
	if err != nil {
		return nil, nil, nil, "", err
	}
	got, err := tail(observed)
	if err != nil {
		return nil, nil, nil, "", err
	}
	var obs any
	if observed != nil {
		obs = observed
	}
	return requested, result, obs, verdict(got, want), nil
}

// tail is the read-back's tail(d): d with sourceOrd cut to `h:<handle>`; nil stays nil.
func tail(d *pyjson.Object) (any, error) {
	if d == nil {
		return nil, nil
	}
	s, err := sourceTail(d)
	if err != nil {
		return nil, err
	}
	out := obj()
	for _, k := range d.Keys() {
		out.Set(k, get(d, k))
	}
	out.Set("sourceOrd", s)
	return out, nil
}
