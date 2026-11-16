package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Rollback is n4_rollback: the recorded inverse of a journaled batch, run as a new batch linked
// to it. A re-created subtree gets one add per nested component and its links back as ops of
// the same batch (Run); the read-back reports what the snapshot could not bring back.
var Rollback = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: rollbackPlan,
	Run: rollbackRuns, Inverse: rollbackInverse, Readback: rollbackReadbackAll}

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
	outgoing                      []any             // the outgoing links a remove recorded (rollback_outgoing.go)
	outgoingSent, outgoingSkipped []any             // filled by the runs for the read-back
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
	outgoing, err := outgoingParts(inverse)
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
	if err := laterWork(c, w, view, ops, targets, bid); err != nil {
		return nil, err
	}
	if err := unchangedSince(c, view, ops, targets); err != nil {
		return nil, err
	}
	if err := flagsUnchangedSince(c, view, ops, targets); err != nil {
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
		o.Set("b", top) // the view is this plan's own copy of the journal line
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
	relinkWrites, outgoingWrites := len(relinks)*len(own), len(outgoing)*len(own)
	if need := 1 + len(components) + relinkWrites + outgoingWrites; w.WritesExecuted+need > w.MaxWrites {
		return nil, plan.ToolError(budgetRefusal(need, len(components), relinkWrites, outgoingWrites,
			w.MaxWrites-w.WritesExecuted, w.MaxWrites))
	}
	if err := checkRestoredEnds(w, targets, ops, own, components, relinks); err != nil {
		return nil, err
	}
	if err := checkOutgoingEnds(w.Scope, targets, own, outgoing); err != nil {
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
	notes := rollbackNotes(view, bid, relinks, unlinked, components, own, ops)
	if len(outgoing) > 0 {
		notes = append(notes, outgoingNote(len(outgoing)))
	}
	return &plan.Planned{Ops: ops, Inverse: planInverse, Notes: notes, Targets: targets, Keys: keys,
		Data: &rollbackData{rollbackOf: bid, targets: targets, relinks: relinks, unlinked: unlinked,
			components: components, ordOf: ordOf, outgoing: outgoing}}, nil
}
