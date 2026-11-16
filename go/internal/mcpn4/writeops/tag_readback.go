package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// setTagsReadback reloads each target once (depth 1): verified when every added tag is a direct
// tag of the sent type and every removed one is gone. A value the station normalized ("4.50"
// stored as "4.5") is reported under coerced, not a mismatch; another value is.
func setTagsReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	verdict, requested, observed := "verified", []any{}, []any{}
	for _, e := range p.Data.([]any) {
		exp := e.(tagExpect)
		_, nodes, err := plan.Handle(c, exp.ord, 1)
		if err != nil {
			return fail(err)
		}
		added, removed, coerced := []any{}, []any{}, []any{}
		want := obj("ord", exp.ord, "add", []any{}, "remove", []any{})
		for _, a := range exp.added {
			slot := a.id.Slot()
			want.Set("add", append(get(want, "add").([]any), obj("id", a.id.String(), "type", get(a.body, "t"), "value", get(a.body, "v"))))
			node := box.Node(nodes, slot)
			if node == nil || !isTag(node, slot) || get(node, "t") != get(a.body, "t") {
				verdict = "mismatch"
				continue
			}
			got := obj("id", a.id.String(), "type", get(node, "t"), "value", get(node, "v"))
			added = append(added, got)
			if !box.PyEqual(get(node, "v"), get(a.body, "v")) {
				if !sameNumber(get(node, "v"), get(a.body, "v")) {
					verdict = "mismatch"
				}
				coerced = append(coerced, obj("id", a.id.String(), "sent", get(a.body, "v"), "stored", get(node, "v")))
			}
		}
		for _, id := range exp.removed {
			want.Set("remove", append(get(want, "remove").([]any), id.String()))
			if box.Node(nodes, id.Slot()) != nil {
				verdict = "mismatch"
				continue
			}
			removed = append(removed, id.String())
		}
		requested = append(requested, want)
		observed = append(observed, obj("ord", exp.ord, "added", added, "removed", removed, "coerced", coerced))
	}
	return requested, replies, observed, verdict, nil
}

// sameNumber is whether two encoded values are the same number written differently.
func sameNumber(a, b any) bool {
	x, okA := a.(string)
	y, okB := b.(string)
	if !okA || !okB {
		return false
	}
	fx, okX := pyjson.ParseFloat(x)
	fy, okY := pyjson.ParseFloat(y)
	return okX && okY && fx == fy
}
