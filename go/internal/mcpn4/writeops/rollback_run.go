package writeops

import (
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// rollbackRuns runs a rollback's follow-up ops in its own batch, after its top-level adds: the
// nested components (_run_components), then the links between restored components
// (_run_relinks). Each op is scope-checked, journaled before it is sent and counted against
// the budget; a failure leaves the batch in-doubt and lists the components that exist now.
func rollbackRuns(b *plan.Batch, p *plan.Planned, replies []any) (*plan.Runs, error) {
	d, runs := p.Data.(*rollbackData), &plan.Runs{}
	if len(d.components) == 0 && len(d.relinks) == 0 && len(d.outgoing) == 0 {
		return runs, nil
	}
	base, err := topCreated(p, d, replies)
	if err != nil {
		return nil, err
	}
	if len(d.components) > 0 {
		if runs.Frozen, err = runComponents(b, d, base); err != nil {
			return nil, err
		}
	}
	if len(d.relinks) > 0 || len(d.outgoing) > 0 {
		if err := runRelinks(b, p, d, base, replies, runs); err != nil {
			return nil, err
		}
	}
	return runs, nil
}

// topCreated is _top_created: {op index: ORD} of the components the batch's own adds created.
func topCreated(p *plan.Planned, d *rollbackData, replies []any) (map[int]string, error) {
	out := map[int]string{}
	for i, op := range p.Ops {
		if o := op.(*pyjson.Object); get(o, "nm") == "a" {
			comp, err := restoredOrd(d, o, replies[i])
			if err != nil {
				return nil, err
			}
			out[i] = comp
		}
	}
	return out, nil
}

// restoredOrd is the ORD of the component an add op created: its parent's ORD and the name the
// station assigned.
func restoredOrd(d *rollbackData, op *pyjson.Object, reply any) (string, error) {
	k, err := pyjson.DictKey(get(op, "h"))
	if err != nil {
		return "", unhashable(get(op, "h"))
	}
	return childOrd(d.ordOf[k], assignedName(op, reply))
}

// assignedName is _assigned_name: the name the station answered ("nn"), else the requested one.
func assignedName(op *pyjson.Object, reply any) any {
	if r, ok := reply.([]any); ok && len(r) > 0 {
		if first, ok := r[0].(*pyjson.Object); ok {
			if nn, ok := first.Get("nn"); ok {
				return nn
			}
		}
	}
	return get(op, "n")
}

// withCreated is _in_doubt_with_created: the in-doubt error of a rollback that stopped
// mid-way, listing the components that exist now.
func withCreated(err *plan.InDoubt, base map[int]string, created []any) error {
	done := make([]string, 0, len(base))
	for _, o := range base {
		done = append(done, o)
	}
	sort.Strings(done) // code point order: WTF-8 bytes sort as code points
	for _, c := range created {
		if c != nil {
			done = append(done, c.(string))
		}
	}
	err.Msg += ". Components already created by this batch (remove the top-level one to clean up): " +
		strings.Join(done, ", ")
	return err
}

// runComponents is _run_components: one add per nested component, parents first, under the
// parent's new handle; a child the fresh parent already has is frozen and never re-added.
func runComponents(b *plan.Batch, d *rollbackData, base map[int]string) ([]any, error) {
	where := map[string]string{} // (top, path) -> ORD
	for i, o := range base {
		where[strconv.Itoa(i)+":"] = o
	}
	frozen, skipped := map[string]map[string]bool{}, map[string]bool{}
	var created, frozenOut []any
	for _, s := range d.components {
		spec := s.(*pyjson.Object)
		top, n := strconv.Itoa(get(spec, "top").(int))+":", get(spec, "n").(string)
		key, here := top+get(spec, "parent_path").(string), top+specPath(spec)
		parent := where[key]
		if !skipped[key] {
			parentH, nodes, err := parentOf(b, parent, n)
			if err != nil {
				return nil, withCreated(plan.InDoubtOf(b.ID, err), base, created)
			}
			if frozen[key] == nil { // what a fresh parent has before any add: its frozen slots
				frozen[key] = map[string]bool{}
				for _, k := range nodes.Keys() {
					frozen[key][k] = k != "" && !strings.Contains(k, "/")
				}
			}
			if !frozen[key][n] {
				op := obj("nm", "a", "h", parentH, "n", n, "b", get(spec, "b"))
				if b.State.Journal.Append(obj("batch_id", b.ID, "ts", session.Now(), "phase", "component-intent",
					"ops", []any{op})) != nil {
					return nil, withCreated(plan.InDoubtBecause(b.ID, "the component intent could not be written, "+
						"nothing more was sent"), base, created)
				}
				b.State.WritesExecuted++
				reply, err := b.Client.Sync(op)
				if err != nil {
					return nil, withCreated(plan.InDoubtOf(b.ID, err), base, created)
				}
				comp, err := childOrd(parent, assignedName(op, reply))
				if err != nil {
					return nil, err
				}
				where[here], created = comp, append(created, comp)
				continue
			}
		}
		skipped[here] = true // a frozen child, or below one: not re-created
		path, err := ord.ChildOrd(parent, n)
		if err != nil {
			return nil, err
		}
		where[here] = path
		body := get(spec, "b").(*pyjson.Object)
		frozenOut = append(frozenOut, obj("path", path, "n", n, "type", get(body, "t")))
		d.frozenChecks = append(d.frozenChecks, obj("path", path, "b", body))
		created = append(created, nil)
	}
	d.created, d.hasCreated = list(created), true
	return frozenOut, nil
}

// parentOf checks the child under today's scope and loads its parent's new handle.
func parentOf(b *plan.Batch, parent, n string) (any, *pyjson.Object, error) {
	child, err := ord.ChildOrd(parent, n)
	if err != nil {
		return nil, nil, err
	}
	if err := b.State.Scope.Check(child); err != nil {
		return nil, nil, err
	}
	return plan.Handle(b.Client, parent, 1)
}
