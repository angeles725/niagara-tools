package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// ReplicatePoints is n4_replicate_points: M points of one parent (e.g. a Modbus device's points)
// copied onto N target parents (other devices' points), N x M adds in one batch under one
// confirmation token (field retro 2026-11-16, row U10). Each copy is n4_copy_component's
// configured body (address and dataType included), with a per-target copies[] name or
// proxyExt/dataAddress override. Every add spends one write of --max-writes: the dry run shows the
// count, and a plan larger than the remaining budget is refused before any token.
var ReplicatePoints = plan.Impl{ScopeOrds: replicaScope, PlanWith: replicaPlan, Inverse: replicaInverse,
	Readback: replicaReadback}

// replica is one planned copy: its target, the target's handle, the requested name, the source
// type and the body the read-back compares.
type replica struct {
	target, name string
	parent, typ  any
	body         *pyjson.Object
}

type replicaData struct {
	copies []replica
	shared map[int]string // the writable register a copy would share, by op index
}

// replicaScope is every target parent: the source is only read.
func replicaScope(args *pyjson.Object) []any {
	raw, _ := args.Get("targets")
	list, _ := raw.([]any)
	out := []any{}
	for _, e := range list {
		out = append(out, get(asObj(e), "parent_ord"))
	}
	return out
}

func replicaPlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	points, targets, err := replicaArgs(args)
	if err != nil {
		return nil, err
	}
	srcOrd := str(args, "source_parent_ord")
	_, srcNodes, err := plan.Handle(c, srcOrd, 1)
	if err != nil {
		return nil, err
	}
	sources, notes := map[string]*copySource{}, []any{}
	for _, p := range points {
		src, err := loadCopySource(c, srcOrd, p, srcNodes)
		if err != nil {
			return nil, err
		}
		sources[p] = src
		notes = append(notes, prefixed(p, strippedNotes(src.stripped))...)
		notes = append(notes, src.linksNote()...)
		notes = append(notes, fallbackNotes(p, src.body)...)
	}
	d := replicaData{shared: map[int]string{}}
	ops, inverse, handles := []any{}, []any{}, pyjson.NewObject()
	for _, t := range targets {
		copies, err := replicaTargetCopies(c, t, points, sources)
		if err != nil {
			return nil, err
		}
		handles.Set(t.ord, copies[0].parent)
		dupNotes, shared := targetRegisters(t, points, sources, srcOrd)
		notes = append(notes, dupNotes...)
		for i, r := range copies {
			if text, ok := shared[points[i]]; ok {
				d.shared[len(d.copies)] = text
			}
			d.copies = append(d.copies, r)
			ops = append(ops, obj("nm", "a", "h", r.parent, "n", r.name, "b", r.body))
			inverse = append(inverse, obj("nm", "v", "h", r.parent, "n", r.name))
		}
	}
	if err := w.CheckRoom(len(ops), "this replication"); err != nil {
		return nil, err
	}
	head := "replicates " + strconv.Itoa(len(points)) + " point(s) onto " + strconv.Itoa(len(targets)) +
		" target(s): " + strconv.Itoa(len(ops)) + " add(s) in one batch under one confirmation token, " +
		strconv.Itoa(len(ops)) + " write(s) of the budget (one per add; write_budget.remaining shows what is left)"
	notes = append([]any{head, "The station may rename on collision; the real inverse uses the assigned name."}, notes...)
	return &plan.Planned{Ops: ops, Inverse: inverse, Notes: notes, Targets: handles, Data: d, Writes: len(ops)}, nil
}

// replicaTargetCopies is the copy of each point under target t, in points order: t's own name and address
// for the point when its copies[] gives them. A name that already exists under t, or that two
// points would share, is refused.
func replicaTargetCopies(c plan.Client, t replicaTarget, points []string, sources map[string]*copySource) ([]replica, error) {
	h, nodes, err := plan.Handle(c, t.ord, 1)
	if err != nil {
		return nil, err
	}
	out, taken := []replica{}, map[string]bool{}
	for _, p := range points {
		n := t.newName(p)
		if _, err := plan.Name("new_name", n); err != nil {
			return nil, plan.ToolError("point " + pyjson.StrRepr(p) + " has a station-escaped name: give its copy " +
				"a plain new_name in copies[] of target " + t.ord)
		}
		if box.Node(nodes, n) != nil || taken[n] {
			return nil, plan.ToolError("a child named " + pyjson.StrRepr(n) + " already exists under " + t.ord +
				" (or two points would share it): give the copy its own new_name in copies[]")
		}
		taken[n] = true
		src, body := sources[p], sources[p].body
		if ov, ok := t.addrOf[p]; ok {
			addrType, err := declaredAddressType(src.ct, src.root)
			if err == nil && addrType == "" {
				err = plan.ToolError("copies[].data_address needs a source with a proxyExt that declares " +
					"dataAddress; " + pyjson.StrRepr(p) + " has none")
			}
			if err != nil {
				return nil, err
			}
			body = withAddress(body, ov, addrType)
		}
		out = append(out, replica{t.ord, n, h, get(src.root, "t"), body})
	}
	return out, nil
}

// prefixed is each note led by the point it is about.
func prefixed(point string, notes []any) []any {
	out := make([]any, len(notes))
	for i, n := range notes {
		out[i] = pyjson.StrRepr(point) + ": " + pyjson.Str(n)
	}
	return out
}
