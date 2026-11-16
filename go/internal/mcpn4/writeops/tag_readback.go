package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// setTagsReadback reloads each target once (depth 1): verified when every added tag is a direct
// tag of the sent type, every removed one is gone, and every relation (tag group) added is there
// and every removed one gone. A value the station normalized ("4.50" stored as "4.5") is reported
// under coerced, not a mismatch; another value is. A target that gained a tag group also reports
// its implied tags (one impliedTags call) and the group tags missing from them.
func setTagsReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	verdict, requested, observed := "verified", []any{}, []any{}
	for _, exp := range p.Data.(*tagData).expects {
		_, nodes, err := plan.Handle(c, exp.ord, 1)
		if err != nil {
			return fail(err)
		}
		want, seen, ok := tagsObserved(nodes, exp)
		if !ok {
			verdict = "mismatch"
		}
		rels := []any{}
		for _, r := range exp.addRels {
			want.Set("add_relations", append(asList(get(want, "add_relations")), r.label))
			if relSlot(nodes, r) == "" {
				verdict = "mismatch"
			} else {
				rels = append(rels, r.label)
			}
		}
		gone := []any{}
		for _, r := range exp.rmRels {
			want.Set("remove_relations", append(asList(get(want, "remove_relations")), r.label))
			if relSlot(nodes, r) != "" {
				verdict = "mismatch"
			} else {
				gone = append(gone, r.label)
			}
		}
		seen.Set("relations_added", rels)
		seen.Set("relations_removed", gone)
		groupImplied(c, exp, seen)
		requested, observed = append(requested, want), append(observed, seen)
	}
	return requested, replies, observed, verdict, nil
}

// tagsObserved checks the tags of one target: what was asked, what is there, and whether it
// matches.
func tagsObserved(nodes *pyjson.Object, exp tagExpect) (want, seen *pyjson.Object, ok bool) {
	ok = true
	added, removed, coerced, wantAdd, wantRm := []any{}, []any{}, []any{}, []any{}, []any{}
	for _, a := range exp.added {
		slot := a.id.Slot()
		wantAdd = append(wantAdd, obj("id", a.id.String(), "type", get(a.body, "t"), "value", get(a.body, "v")))
		node := box.Node(nodes, slot)
		if node == nil || !isTag(node, slot) || get(node, "t") != get(a.body, "t") {
			ok = false
			continue
		}
		added = append(added, obj("id", a.id.String(), "type", get(node, "t"), "value", get(node, "v")))
		if !box.PyEqual(get(node, "v"), get(a.body, "v")) {
			ok = ok && sameNumber(get(node, "v"), get(a.body, "v"))
			coerced = append(coerced, obj("id", a.id.String(), "sent", get(a.body, "v"), "stored", get(node, "v")))
		}
	}
	for _, id := range exp.removed {
		wantRm = append(wantRm, id.String())
		if box.Node(nodes, id.Slot()) != nil {
			ok = false
			continue
		}
		removed = append(removed, id.String())
	}
	return obj("ord", exp.ord, "add", wantAdd, "remove", wantRm),
		obj("ord", exp.ord, "added", added, "removed", removed, "coerced", coerced), ok
}

// groupImplied adds to seen the implied tags of a target that gained a tag group, and the group
// tags not among them; informational: a failed call is reported, never a mismatch.
func groupImplied(c plan.Client, exp tagExpect, seen *pyjson.Object) {
	var wanted []string
	for _, r := range exp.addRels {
		if r.group {
			wanted = append(wanted, r.implies...)
		}
	}
	if len(wanted) == 0 {
		return
	}
	reply, err := c.SSC("impliedTags", exp.h)
	m, isObj := reply.(*pyjson.Object)
	if err != nil || !isObj {
		seen.Set("implied_unavailable", "impliedTags: "+pyjson.Repr(reply)+" "+pyjson.Repr(errText(err)))
		return
	}
	have, ids := map[string]bool{}, []any{}
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		if tag, ok := tags.DecodeImplied(k, v); ok { // the key ends in its type letter (n:pointM)
			have[tag.ID], ids = true, append(ids, tag.ID)
		}
	}
	missing := []any{}
	for _, w := range wanted {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	seen.Set("implied", ids)
	seen.Set("group_tags_missing", missing)
}

func errText(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
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
