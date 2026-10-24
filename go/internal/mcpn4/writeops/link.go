package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// CreateLink is n4_create_link: one checkLinks call that makes the link; the inverse removes
// the link the station named. A reply that cannot be trusted is in-doubt, and Recover looks
// for the new link by its slots.
var CreateLink = plan.Impl{ScopeOrds: func(args *pyjson.Object) []any { return []any{str(args, "source_ord"), str(args, "target_ord")} },
	Plan: linkPlan, Inverse: linkInverse, Recover: linkRecover, Readback: linkReadback}

type linkData struct {
	srcH, tgtH any
	existing   map[string]bool // the links the target had when planned
}

func linkPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	ssArg, _ := args.Get("source_slot")
	ss, err := plan.Name("source_slot", ssArg)
	if err != nil {
		return nil, err
	}
	tsArg, _ := args.Get("target_slot")
	ts, err := plan.Name("target_slot", tsArg)
	if err != nil {
		return nil, err
	}
	srcH, _, err := plan.Handle(c, str(args, "source_ord"), 1)
	if err != nil {
		return nil, err
	}
	tgtOrd := str(args, "target_ord")
	tgtH, nodes, err := plan.Handle(c, tgtOrd, 1)
	if err != nil {
		return nil, err
	}
	data := linkData{srcH: srcH, tgtH: tgtH, existing: map[string]bool{}}
	for _, name := range links(nodes) {
		data.existing[name] = true
	}
	targets := pyjson.NewObject()
	targets.Set(tgtOrd, tgtH)
	return &plan.Planned{
		Ops:     []any{obj("ssc", "checkLinks", "arg", obj("s", srcH, "ss", ss, "t", tgtH, "ts", ts, "c", true))},
		Inverse: []any{obj("nm", "v", "h", tgtH, "n", "<link name assigned by the station>")},
		Notes:   []any{}, Targets: targets, Data: data}, nil
}

// links is the names of the baja:Link children of a loaded component, in load order.
func links(nodes *pyjson.Object) []string {
	var out []string
	kids := box.Children(nodes, "")
	for _, name := range kids.Keys() {
		if t, _ := get(box.Node(kids, name), "t").(string); t == "baja:Link" {
			out = append(out, name)
		}
	}
	return out
}

// linkResult is _link_result: the verdict dict of a checkLinks reply.
func linkResult(replies []any) (*pyjson.Object, error) {
	if res, ok := replies[0].([]any); ok && len(res) > 0 {
		if inner, ok := res[0].([]any); ok && len(inner) > 0 {
			if d, ok := inner[0].(*pyjson.Object); ok {
				return d, nil
			}
		}
	}
	return nil, linkError("malformed checkLinks reply")
}

func linkError(msg string) error {
	return &box.Error{Kind: box.KindBox, Msg: msg, Channel: box.Channel, Key: "checkLinks"}
}

// linkInverse is _link_inverse: [] for an explicit refusal; an error when the reply cannot be
// trusted.
func linkInverse(p *plan.Planned, replies []any) ([]any, error) {
	result, err := linkResult(replies)
	if err != nil {
		return nil, err
	}
	v, ok := result.Get("v")
	switch {
	case !ok:
		return nil, linkError("ambiguous checkLinks reply: no verdict")
	case !pyjson.Truthy(v):
		return []any{}, nil // the station refused the link: nothing changed
	case !pyjson.Truthy(get(result, "s")):
		return nil, linkError("ambiguous checkLinks reply: link accepted but not named")
	}
	return []any{obj("nm", "v", "h", p.Data.(linkData).tgtH, "n", get(result, "s"))}, nil
}

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
