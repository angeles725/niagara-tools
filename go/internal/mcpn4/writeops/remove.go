package writeops

import (
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// RemoveComponent is n4_remove_component: one remove of the child name of parent_ord. The
// inverse is a data-only snapshot of the subtree (SnapshotDepth levels) that re-creates it,
// then a relink per link inside it and an unlinked_input per input linked from outside. The
// server runs PlanWith, which always scans for links leaving the subtree (remove_links.go).
var RemoveComponent = plan.Impl{ScopeOrds: removeScope, Plan: removePlan, PlanWith: removePlanWith,
	Readback: removeReadback}

func removeScope(args *pyjson.Object) []any {
	out := []any{str(args, "parent_ord")}
	if v, ok := args.Get("link_scan_ord"); ok {
		out = append(out, v)
	}
	return out
}

// removePlan is _remove_plan.
func removePlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	return removePlanIn(c, args, nil)
}

// removePlanIn is the remove plan; a nil scope is the Python port's link step (scanNotes), else
// the scan always runs (linkScan).
func removePlanIn(c plan.Client, args *pyjson.Object, scope *ord.Scope) (*plan.Planned, error) {
	nameArg, _ := args.Get("name")
	nm, err := plan.ExistingName("name", nameArg)
	if err != nil {
		return nil, err
	}
	parentOrd := str(args, "parent_ord")
	parentH, parentNodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, err
	}
	if _, ok := parentNodes.Get(nm); !ok {
		return nil, plan.ToolError("component " + pyjson.StrRepr(nm) + " not found under " + parentOrd)
	}
	childOrd, err := ord.ChildOrd(parentOrd, nm)
	if err != nil {
		return nil, err
	}
	// One level more than the snapshot keeps: what sits there is reported, never captured.
	nodes, holder, plain, err := childNodes(c, parentH, nm, childOrd)
	if err != nil {
		return nil, err
	}
	var truncated []string
	for _, p := range nodes.Keys() {
		if scope != nil && linkSlot(nodes, p) { // read into the link's record (wholeLinks)
			continue
		}
		if p != "" && strings.Count(p, "/") >= SnapshotDepth {
			truncated = append(truncated, p)
		}
	}
	sort.Strings(truncated) // code point order: WTF-8 bytes sort as code points
	ct, contractsNote := snapshotContracts(c, box.Node(nodes, ""))
	s := &snapshot{handles: map[string]string{}, ct: ct, wholeLinks: scope != nil}
	body, err := s.take(box.Node(nodes, ""), "", SnapshotDepth)
	if err != nil {
		return nil, err
	}
	inverse := []any{slotAdd(parentH, nm, body, box.Node(nodes, ""), plain)}
	notes := []any{"Automatic re-creation (n4_rollback) is limited to this snapshot: type, plain slots " +
		"and wsAnnotation to depth 3; it is not a full restore."}
	if contractsNote != "" {
		notes = append(notes, contractsNote)
	}
	slot, anyPrior := "", false
	if plain && scope != nil { // the Python port's path (no scope) keeps its handles
		slot = nm
		if inverse, notes, anyPrior, err = plainSlotLinks(holder, body, nm, parentH, s.handles, inverse, notes); err != nil {
			return nil, err
		}
	}
	for _, l := range s.links {
		tail, err := sourceTail(obj("sourceOrd", l.sourceOrd))
		if err != nil {
			return nil, err
		}
		source, found := "", false
		if strings.HasPrefix(tail, "h:") {
			source, found = s.handles["s"+tail[2:]]
		}
		spec := obj("source_slot", l.sourceSlot, "target_path", l.targetPath, "target_slot", l.targetSlot)
		if l.prior != nil { // the input's last value, reported if it cannot be relinked
			spec.Set("prior", l.prior)
			anyPrior = true
		}
		if !found {
			where, suffix := l.targetPath, ""
			if where == "" {
				where = nm
			}
			if l.prior != nil {
				suffix = "; its last value is recorded and reported by n4_rollback"
			}
			notes = append(notes, "link into "+pyjson.StrRepr(where)+" ("+pyjson.Str(l.targetSlot)+" <- "+
				pyjson.Str(l.sourceSlot)+") comes from outside the removed subtree: not restored"+suffix)
			inverse = append(inverse, obj("unlinked_input", spec))
			continue
		}
		spec.Set("source_path", source)
		inverse = append(inverse, obj("relink", spec))
	}
	if anyPrior {
		notes = append(notes, "link-driven input values are captured when the confirmed remove runs, "+
			"not at this dry run (link_input_values is a preview) and are not part "+
			"of the confirmation hash; they are never written back: n4_rollback "+
			"reports them when it cannot re-establish their link")
	}
	targets, keys := pyjson.NewObject(), pyjson.NewObject()
	targets.Set(parentOrd, parentH)
	if len(truncated) > 0 { // audit 2026-10-03 F8: the depth cut used to be silent
		shown, listed := truncated, ""
		if len(truncated) > TruncatedPathsShown {
			shown, listed = truncated[:TruncatedPathsShown], ", first "+strconv.Itoa(TruncatedPathsShown)+" listed"
		}
		paths := make([]any, len(shown))
		for i, p := range shown {
			paths[i] = p
		}
		keys.Set("snapshot_truncated", obj("depth", pyjson.IntLiteral(strconv.Itoa(SnapshotDepth)), "paths", paths))
		notes = append(notes, "the subtree is deeper than the snapshot (depth 3): at least "+strconv.Itoa(len(truncated))+
			" slot(s) or component(s) below it are NOT captured (only the first level below the cut is loaded, "+
			"so deeper ones are not counted) and a rollback cannot bring them back (see snapshot_truncated"+
			listed+"); export or note them before removing")
	}
	if scope == nil {
		notes, err = scanNotes(c, args, nodes, s.handles, keys, notes)
	} else {
		notes, inverse, err = linkScan(c, args, scope, nodes, s.handles, slot, keys, notes, inverse)
	}
	if err != nil {
		return nil, err
	}
	p := &plan.Planned{Ops: []any{obj("nm", "v", "h", parentH, "n", nm)}, Inverse: inverse, Notes: notes,
		Targets: targets, Keys: keys}
	first, ok := firstObj(inverse)
	if !ok { // the snapshot always yields the re-add first; a broken invariant is a tool error, never a panic
		return nil, plan.ToolError("internal: the remove plan of " + pyjson.StrRepr(nm) + " has no re-add op to hash")
	}
	if add := s.hashedAdd(first); add != nil { // retro 2026-11-16 row M15
		p.HashInverse = append([]any{add}, inverse[1:]...)
		p.Notes = append(p.Notes, msgRuntimeUnhashed)
	}
	return p, nil
}

// removeReadback is _remove_readback: the child must be gone from its parent.
func removeReadback(c plan.Client, args *pyjson.Object, _ *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	nm := str(args, "name")
	_, still := nodes.Get(nm)
	v := "verified"
	if still {
		v = "mismatch"
	}
	return obj("removed", nm), replies[0], obj("gone", !still), v, nil
}

// firstObj is the first op of ops when it is an object.
func firstObj(ops []any) (*pyjson.Object, bool) {
	if len(ops) == 0 {
		return nil, false
	}
	o, ok := ops[0].(*pyjson.Object)
	return o, ok && o != nil
}
