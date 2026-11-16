package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// SetTags is n4_set_tags: direct tags added (a metadata slot, flag m), changed (same type) or
// removed, and tag groups and relations added or removed, on 1 to 50 components, after the
// installed dictionaries validate them.
var SetTags = plan.Impl{ScopeOrds: tagOrdsArg, Plan: setTagsPlan, Inverse: setTagsInverse, Readback: setTagsReadback}

// tagExpect is what the read-back checks on one target: the tags each add sent, the tags removed,
// and the relations (tag groups included) added and removed.
type tagExpect struct {
	ord             string
	h               any
	added           []tagAdd
	removed         []tags.ID
	addRels, rmRels []tagRel
}

// tagData is the plan's private data: the read-back's expectations and the inverse of each op in
// op order (an added relation's slot name comes from the station's reply).
type tagData struct {
	expects []tagExpect
	fwd     []any
}

// tagRequest is every change the arguments ask for, resolved once for all targets.
type tagRequest struct {
	adds                                 []tagAdd
	removes                              []tags.ID
	addGroups, rmGroups, addRels, rmRels []tagRel
}

func (r *tagRequest) count() int {
	return len(r.adds) + len(r.removes) + len(r.addGroups) + len(r.rmGroups) + len(r.addRels) + len(r.rmRels)
}

// resolveTags reads the dictionaries (once, only when an add, group or relation needs them).
func resolveTags(c plan.Client, args *pyjson.Object) (*tagRequest, error) {
	r := &tagRequest{}
	var err error
	if r.removes, err = tagRemoves(args); err != nil {
		return nil, err
	}
	needs := false
	for _, k := range []string{"add", "add_groups", "remove_groups", "add_relations", "remove_relations"} {
		needs = needs || len(asList(lookupArg(args, k))) > 0
	}
	if !needs {
		return r, nil
	}
	cat, err := tags.LoadCatalog(c)
	if err != nil {
		return nil, err
	}
	if r.adds, err = tagAdds(args, cat); err != nil {
		return nil, err
	}
	if r.addGroups, err = tagGroups(args, "add_groups", cat); err != nil {
		return nil, err
	}
	if r.rmGroups, err = tagGroups(args, "remove_groups", cat); err != nil {
		return nil, err
	}
	if r.addRels, err = tagRelations(c, args, "add_relations", cat); err != nil {
		return nil, err
	}
	r.rmRels, err = tagRelations(c, args, "remove_relations", cat)
	return r, err
}

func setTagsPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	ords, err := tagTargets(args)
	if err != nil {
		return nil, err
	}
	req, err := resolveTags(c, args)
	if err != nil {
		return nil, err
	}
	if n := req.count() * len(ords); n == 0 {
		return nil, plan.ToolError("nothing to change: give add, remove, add_groups, remove_groups, add_relations or remove_relations")
	} else if n > MaxTagOps {
		return nil, plan.ToolError(fmt.Sprintf("the batch would send %d ops (targets x changes); at most %d: split it", n, MaxTagOps))
	}
	p := &plan.Planned{Ops: []any{}, Notes: []any{}, Targets: pyjson.NewObject()}
	d := &tagData{}
	for _, o := range ords {
		exp, err := planTarget(c, o, req, p, d)
		if err != nil {
			return nil, err
		}
		d.expects = append(d.expects, exp)
	}
	if len(p.Ops) == 0 {
		why := []string{}
		for _, n := range p.Notes {
			why = append(why, pyjson.Str(n))
		}
		return nil, plan.ToolError("nothing to change: every tag, group and relation is already as requested (" +
			strings.Join(why, "; ") + ")")
	}
	p.Inverse, p.Data = reversed(d.fwd), d
	return p, nil
}

// planTarget appends the ops of one target to p and their inverses to d.fwd.
func planTarget(c plan.Client, o string, req *tagRequest, p *plan.Planned, d *tagData) (tagExpect, error) {
	h, nodes, err := plan.Handle(c, o, 1)
	if err != nil {
		return tagExpect{}, err
	}
	p.Targets.Set(o, h)
	exp := tagExpect{ord: o, h: h}
	push := func(op, inv *pyjson.Object) { p.Ops, d.fwd = append(p.Ops, op), append(d.fwd, inv) }
	for _, a := range req.adds {
		op, inv, note, err := tagAddOp(h, nodes, a, o)
		if err != nil {
			return exp, err
		}
		if note != "" {
			p.Notes = append(p.Notes, note)
		} else {
			push(op, inv)
			exp.added = append(exp.added, a)
		}
	}
	for _, id := range req.removes {
		op, inv, err := tagRemoveOp(h, nodes, id, o)
		if err != nil {
			return exp, err
		}
		push(op, inv)
		exp.removed = append(exp.removed, id)
	}
	for _, r := range append(append([]tagRel{}, req.addGroups...), req.addRels...) {
		op, inv, note := tagRelAddOp(h, nodes, r, o)
		if note != "" {
			p.Notes = append(p.Notes, note)
		} else {
			push(op, inv)
			exp.addRels = append(exp.addRels, r)
		}
	}
	for _, r := range append(append([]tagRel{}, req.rmGroups...), req.rmRels...) {
		op, inv, err := tagRelRemoveOp(h, nodes, r, o)
		if err != nil {
			return exp, err
		}
		push(op, inv)
		exp.rmRels = append(exp.rmRels, r)
	}
	return exp, nil
}

// setTagsInverse is the journaled inverse: each op's inverse, last op first, an added relation's
// remove naming the slot the station answered ("nn"), else the planned "r?".
func setTagsInverse(p *plan.Planned, replies []any) ([]any, error) {
	fwd := append([]any{}, p.Data.(*tagData).fwd...)
	for i, e := range fwd {
		inv := e.(*pyjson.Object)
		if get(inv, "nm") != "v" || get(inv, "n") != "r?" || i >= len(replies) {
			continue
		}
		if first, ok := replies[i].([]any); ok && len(first) > 0 {
			if reply, ok := first[0].(*pyjson.Object); ok {
				if nn, ok := reply.Get("nn"); ok {
					fwd[i] = obj("nm", "v", "h", get(inv, "h"), "n", nn)
				}
			}
		}
	}
	return reversed(fwd), nil
}

func reversed(list []any) []any {
	out := make([]any, 0, len(list))
	for i := len(list) - 1; i >= 0; i-- {
		out = append(out, list[i])
	}
	return out
}

func asList(v any) []any { l, _ := v.([]any); return l }
