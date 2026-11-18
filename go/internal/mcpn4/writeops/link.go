package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

type linkData struct {
	srcH, tgtH any
	existing   map[string]bool // the links the target had when planned
}

// linkPlan plans one link: the single form's arguments, or one entry of links (same keys).
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
	srcH, srcNodes, err := plan.Handle(c, str(args, "source_ord"), 1)
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
	if err := linkPrecheck(c, args, srcNodes, nodes, data, ss, ts); err != nil {
		return nil, err
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
