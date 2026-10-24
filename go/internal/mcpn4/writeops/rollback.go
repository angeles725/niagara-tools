package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Rollback is n4_rollback: the recorded inverse of a journaled batch, run as a new batch linked
// to it. A re-created subtree gets one add per nested component and its links back as ops of
// the same batch (Run); the read-back reports what the snapshot could not bring back.
var Rollback = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: rollbackPlan,
	Run: rollbackRuns, Inverse: rollbackInverse, Readback: rollbackReadback}

// batchJournal is what the rollback reads of the journal (journal.Journal).
type batchJournal interface {
	RollbackTarget(batchID any) (*pyjson.Object, error)
	EarlierRollbacks(batchID string) error
}

// rollbackData is the plan's private data; the runs fill created, the frozen checks, the
// skipped relinks and the relink report for the read-back.
type rollbackData struct {
	rollbackOf                    string
	targets                       *pyjson.Object
	relinks, unlinked, components []any
	ordOf                         map[string]string // handle (as a dict key) -> ORD
	created, frozenChecks         []any             // created: an ORD per spec, nil when skipped
	hasCreated                    bool
	skipped                       []any
	report                        *pyjson.Object
}

// rollbackPlan is _rollback_plan: the journal checks, the station check of every recorded
// target under today's scope, the unchanged-since check of restored slots, then one add per
// nested component, the budget and the scope of every re-created component and relink end.
func rollbackPlan(c plan.Client, args *pyjson.Object, w *plan.State, sess *session.Session) (*plan.Planned, error) {
	j, ok := w.Journal.(batchJournal)
	if !ok {
		return nil, &plan.PyError{Type: "AttributeError", Msg: "the journal cannot be read"}
	}
	bidArg, _ := args.Get("batch_id")
	view, err := j.RollbackTarget(bidArg)
	if err != nil {
		return nil, err
	}
	bid := bidArg.(string)
	if st := get(view, "station_name"); !box.PyEqual(st, sess.StationName) {
		return nil, plan.ToolError(fmt.Sprintf("batch %s was recorded on station %s, this session is on %s",
			bid, pyjson.Repr(st), pyjson.Repr(sess.StationName)))
	}
	if err := j.EarlierRollbacks(bid); err != nil {
		return nil, err
	}
	inverse, err := items(orEmpty(get(view, "inverse")))
	if err != nil {
		return nil, err
	}
	ops, relinks, unlinked, err := inverseParts(inverse)
	if err != nil {
		return nil, err
	}
	if len(ops) == 0 {
		return nil, plan.ToolError("batch " + bid + " has no inverse recorded (verdict " + pyjson.Str(get(view, "verdict")) +
			"): nothing to roll back")
	}
	targets, err := recordedTargets(view, ops, bid)
	if err != nil {
		return nil, err
	}
	for _, o := range targets.Keys() {
		if err := w.Scope.Check(o); err != nil { // today's scope, not the one the batch ran under
			return nil, err
		}
		current, _, err := plan.Handle(c, o, 1)
		if err != nil {
			return nil, err
		}
		if recorded := get(targets, o); !box.PyEqual(current, recorded) {
			return nil, plan.ToolError(o + " now has handle " + pyjson.Str(current) + " but the batch recorded " +
				pyjson.Str(recorded) + ": the component was replaced, refusing")
		}
	}
	if err := unchangedSince(c, view, ops, targets); err != nil {
		return nil, err
	}
	var components, own []any
	for i, op := range ops { // one add per component: nested bodies are rejected live
		o := op.(*pyjson.Object)
		if get(o, "nm") != "a" {
			continue
		}
		b, err := need(o, "b")
		if err != nil {
			return nil, err
		}
		top, specs, err := flatten(i, b)
		if err != nil {
			return nil, err
		}
		o = copyObj(o)
		o.Set("b", top)
		ops[i], components = o, append(components, specs...)
	}
	for _, op := range ops {
		if o := op.(*pyjson.Object); get(o, "nm") == "a" {
			h, err := need(o, "h")
			if err != nil {
				return nil, err
			}
			n, err := need(o, "n")
			if err != nil {
				return nil, err
			}
			own = append(own, obj("nm", "v", "h", h, "n", n))
		}
	}
	links := len(relinks) * len(own)
	if need := 1 + len(components) + links; w.WritesExecuted+need > w.MaxWrites {
		return nil, plan.ToolError(fmt.Sprintf("%s: this rollback needs %d write(s) (1 batch + %d component(s) + %d "+
			"relink(s)) but only %d remain (--max-writes %d)", safety.ReasonBudgetSmall, need, len(components), links,
			w.MaxWrites-w.WritesExecuted, w.MaxWrites))
	}
	if err := checkRestoredEnds(w, targets, ops, own, components, relinks); err != nil {
		return nil, err
	}
	ordOf := map[string]string{}
	for _, o := range targets.Keys() {
		k, err := pyjson.DictKey(get(targets, o))
		if err != nil {
			return nil, unhashable(get(targets, o))
		}
		ordOf[k] = o
	}
	planInverse := []any{}
	if len(own) == len(ops) {
		planInverse = own
	}
	keys := obj("rollback_of", bid, "relinks", list(relinks), "unlinked_inputs", list(unlinked), "components", list(components))
	return &plan.Planned{Ops: ops, Inverse: planInverse, Notes: rollbackNotes(view, bid, relinks, unlinked, components, own, ops),
		Targets: targets, Keys: keys, Data: &rollbackData{rollbackOf: bid, targets: targets, relinks: relinks,
			unlinked: unlinked, components: components, ordOf: ordOf}}, nil
}

// inverseParts splits a recorded inverse: its ops (dicts with "nm"), then the validated relink
// specs, then the validated unlinked inputs, each in the order Python's comprehensions check them.
func inverseParts(inverse []any) (ops, relinks, unlinked []any, err error) {
	for _, kind := range []string{"nm", "relink", "unlinked_input"} {
		for _, e := range inverse {
			o, ok := e.(*pyjson.Object)
			if !ok {
				continue
			}
			v, has := o.Get(kind)
			switch {
			case !has:
			case kind == "nm":
				ops = append(ops, o)
			case kind == "relink":
				spec, err := relinkSpec(v)
				if err != nil {
					return nil, nil, nil, err
				}
				relinks = append(relinks, spec)
			default:
				spec, err := unlinkedSpec(v)
				if err != nil {
					return nil, nil, nil, err
				}
				unlinked = append(unlinked, spec)
			}
		}
	}
	return ops, relinks, unlinked, nil
}

// recordedTargets is `view.get("targets") or {}`, refused when it is empty or misses the handle
// of an op.
func recordedTargets(view *pyjson.Object, ops []any, bid string) (*pyjson.Object, error) {
	var targets *pyjson.Object
	if v := get(view, "targets"); pyjson.Truthy(v) {
		t, err := dict(v, "values")
		if err != nil {
			return nil, err
		}
		targets = t
	}
	refused := targets == nil
	for _, op := range ops {
		if refused {
			break
		}
		refused = ordOf(targets, get(op.(*pyjson.Object), "h")) == ""
	}
	if refused {
		return nil, plan.ToolError("batch " + bid + " does not record which components its inverse touches: refusing")
	}
	return targets, nil
}

// checkRestoredEnds checks every re-created component and both ends of every relink of every
// restored component (named as requested) under today's scope.
func checkRestoredEnds(w *plan.State, targets *pyjson.Object, ops, own, components, relinks []any) error {
	for _, s := range components {
		spec := s.(*pyjson.Object)
		top := ops[get(spec, "top").(int)].(*pyjson.Object)
		comp, err := childOrd(ordOf(targets, get(top, "h")), get(top, "n"))
		if err != nil {
			return err
		}
		end, err := ord.JoinOrd(comp, specPath(spec))
		if err != nil {
			return err
		}
		if err := w.Scope.Check(end); err != nil {
			return err
		}
	}
	for _, op := range own {
		comp, err := childOrd(ordOf(targets, get(op.(*pyjson.Object), "h")), get(op.(*pyjson.Object), "n"))
		if err != nil {
			return err
		}
		for _, spec := range relinks {
			if err := checkEnds(w.Scope, comp, spec.(*pyjson.Object)); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkEnds is _check_ends: both ends of a relink under scope.
func checkEnds(scope *ord.Scope, comp string, spec *pyjson.Object) error {
	for _, k := range []string{"source_path", "target_path"} {
		end, err := ord.JoinOrd(comp, get(spec, k).(string))
		if err != nil {
			return err
		}
		if err := scope.Check(end); err != nil {
			return err
		}
	}
	return nil
}

// childOrd is box.child_ord of a parent ORD and a recorded name of any type.
func childOrd(parent string, name any) (string, error) {
	s, ok := name.(string)
	if !ok {
		return "", &pyjson.ValueError{Msg: "ORD child name must be one plain slot name, got " + pyjson.Repr(name)}
	}
	return ord.ChildOrd(parent, s)
}

func specPath(spec *pyjson.Object) string {
	n, parent := get(spec, "n").(string), get(spec, "parent_path").(string)
	if parent == "" {
		return n
	}
	return parent + "/" + n
}

func rollbackNotes(view *pyjson.Object, bid string, relinks, unlinked, components, own, ops []any) []any {
	notes := []any{"rolls back batch " + bid + " (" + pyjson.Str(get(view, "tool")) + ")"}
	if len(relinks) > 0 {
		notes = append(notes, fmt.Sprintf("then re-creates %d link(s) between restored components, where both "+
			"ends exist; a skipped one is reported in link_inputs_not_restored with "+
			"the input's recorded value", len(relinks)))
	}
	if len(unlinked) > 0 {
		notes = append(notes, fmt.Sprintf("%d input(s) were linked from outside the removed subtree: their links "+
			"are not re-created; they are reported in link_inputs_not_restored with "+
			"their recorded value and the verdict is partial", len(unlinked)))
	}
	if len(components) > 0 {
		notes = append(notes, fmt.Sprintf("re-creates %d nested component(s) one add per component, parents "+
			"first: the station rejects an add that nests components", len(components)),
			"a child that already exists on a freshly created component is a frozen "+
				"slot (e.g. a writable's proxyExt): it is never re-added and is listed "+
				"as frozen_children_not_restored; its configuration is not written. The "+
				"read-back compares it with the snapshot and, when it differs, lists it "+
				"in frozen_config_not_restored and the verdict is partial")
	}
	if len(own) == 0 || len(own) != len(ops) {
		notes = append(notes, "the rollback itself has no automatic inverse for slot restores or removals")
	}
	return notes
}

// relinkSpec is _relink_spec: a journaled relink, paths and slot names only, never handles.
func relinkSpec(v any) (*pyjson.Object, error) {
	spec, ok := v.(*pyjson.Object)
	keys := []string{"source_path", "source_slot", "target_path", "target_slot"}
	for _, k := range keys {
		if _, isStr := get(spec, k).(string); !ok || !isStr {
			return nil, plan.ToolError("batch records a malformed relink: refusing")
		}
	}
	out := obj()
	for _, k := range keys {
		s := get(spec, k).(string)
		if s != "" {
			for _, part := range strings.Split(s, "/") {
				if _, err := plan.Name(k, part); err != nil {
					return nil, err
				}
			}
		}
		out.Set(k, s)
	}
	if prior, ok := get(spec, "prior").(*pyjson.Object); ok { // journaled since v0.28.0, absent before
		out.Set("prior", prior)
	}
	return out, nil
}

// unlinkedSpec is _unlinked_spec: an input whose link came from outside the removed subtree.
func unlinkedSpec(v any) (*pyjson.Object, error) {
	spec, ok := v.(*pyjson.Object)
	if !ok {
		return nil, plan.ToolError("batch records a malformed unlinked input: refusing")
	}
	withSource := copyObj(spec)
	withSource.Set("source_path", "")
	out, err := relinkSpec(withSource)
	if err != nil {
		return nil, err
	}
	trimmed := obj()
	for _, k := range out.Keys()[1:] {
		trimmed.Set(k, get(out, k))
	}
	return trimmed, nil
}

func copyObj(o *pyjson.Object) *pyjson.Object {
	c := obj()
	for _, k := range o.Keys() {
		c.Set(k, get(o, k))
	}
	return c
}

func list(v []any) []any {
	if v == nil {
		return []any{}
	}
	return v
}
