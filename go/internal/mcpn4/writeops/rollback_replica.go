package writeops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// The rollback of an in-doubt n4_replicate_points batch (retro 2026-11-17 mcpn4-replicate-points,
// Δ5). Add k of a replication failed after adds 1..k-1 were accepted: the journal holds their
// replies. n4_rollback removes only those copies, by the name the station's reply assigned, when the
// read-back finds them with the type the batch added. The failed add may or may not have been
// applied and a component under its name may not be this batch's, so it is never removed
// ("unverified"); the adds after it were never sent ("not_sent"). Every copy carries its verdict.

// replicateTool is the bulk tool whose rollback this file plans.
const replicateTool = "n4_replicate_points"

// The per-copy verdicts of the plan; a "found" copy reads back as "removed" or "still_present".
const (
	copyFound      = "found"      // accepted by the station and found as added: removed
	copyGone       = "gone"       // accepted, no longer under its target: nothing to remove
	copyChanged    = "changed"    // accepted, its name now holds another type: kept
	copyUnverified = "unverified" // the add that failed: maybe applied, never removed blindly
	copyNotSent    = "not_sent"   // after the failed add: never sent, a same-name one is not ours
	copyLaterWork  = "later_work" // found as added, but a later batch wrote under it: kept
)

// viewReader is the journal's merged view of one batch (journal.Journal.Read).
type viewReader interface {
	Read(batchID string) (*pyjson.Object, error)
}

// inDoubtReplicaOr is the rollback plan of an in-doubt replication whose result record names the
// add that failed; any other refusal of the journal, refused, stays as it is.
func inDoubtReplicaOr(c plan.Client, w *plan.State, sess *session.Session, j batchJournal, bidArg any, refused error) (*plan.Planned, error) {
	reader, ok := j.(viewReader)
	bid, isStr := bidArg.(string)
	if !ok || !isStr || !strings.Contains(refused.Error(), "is in-doubt (the station may or may not have applied it)") {
		return nil, refused
	}
	view, err := reader.Read(bid)
	if err != nil || view == nil || get(view, "state") != "in-doubt" || get(view, "tool") != replicateTool ||
		pyjson.Truthy(get(view, "rollback_of")) {
		return nil, refused
	}
	accepted, isList := get(view, "accepted").([]any)
	failed, partial := view.Get("failed_op") // the add that failed is the one after the accepted ones
	if !partial || !isList || pyjson.Repr(failed) != strconv.Itoa(len(accepted)) {
		return nil, refused // no consistent record of what was accepted
	}
	if st := get(view, "station_name"); !box.PyEqual(st, sess.StationName) {
		return nil, plan.ToolError(fmt.Sprintf("batch %s was recorded on station %s, this session is on %s",
			bid, pyjson.Repr(st), pyjson.Repr(sess.StationName)))
	}
	if err := j.EarlierRollbacks(bid); err != nil {
		return nil, err
	}
	return replicaUndoPlan(c, w, view, bid, accepted)
}

// replicaUndoPlan reads each target once and plans one remove per copy found as added.
func replicaUndoPlan(c plan.Client, w *plan.State, view *pyjson.Object, bid string, accepted []any) (*plan.Planned, error) {
	targets, _ := get(view, "targets").(*pyjson.Object)
	ops, _ := get(view, "ops").([]any)
	if targets == nil || len(ops) == 0 {
		return nil, plan.ToolError("batch " + bid + " records no copies: nothing to roll back")
	}
	loaded, used := map[string]*pyjson.Object{}, pyjson.NewObject()
	removes, copies, notes := []any{}, []any{}, []any{}
	for i, e := range ops {
		op := asObj(e)
		where := ordOf(targets, get(op, "h"))
		name := get(op, "n")
		if i < len(accepted) {
			name = replicaName(accepted[i], pyjson.Str(name))
		}
		state := copyNotSent
		switch {
		case i == len(accepted):
			state = copyUnverified
		case i < len(accepted):
			nodes, err := replicaTargetNodes(c, w, targets, where, loaded)
			if err != nil {
				return nil, err
			}
			state = acceptedCopyState(nodes, name, get(asObj(get(op, "b")), "t"))
			if state == copyFound {
				if state, err = laterWorkState(w, view, bid, where, name); err != nil {
					return nil, err
				}
			}
		}
		entry := obj("op", i+1, "target", where, "name", name, "state", state)
		copies, notes = append(copies, entry), append(notes, copyNote(entry))
		if state == copyFound {
			removes = append(removes, obj("nm", "v", "h", get(op, "h"), "n", name))
			used.Set(where, get(targets, where))
		}
	}
	if len(removes) == 0 {
		return nil, plan.ToolError("batch " + bid + " is in-doubt and none of its copies was accepted by the station " +
			"and found as added, so there is nothing this rollback can prove it created; inspect by hand: " +
			strings.Join(strs(notes), "; "))
	}
	head := fmt.Sprintf("rolls back in-doubt batch %s (%s): removes the %d copy(ies) the station accepted and the "+
		"read-back finds as added; the add that failed may or may not have been applied and is never removed "+
		"blindly: check it and n4_remove_component it by hand", bid, replicateTool, len(removes))
	ordOfH := map[string]string{}
	for _, o := range used.Keys() {
		k, _ := pyjson.DictKey(get(used, o))
		ordOfH[k] = o
	}
	return &plan.Planned{Ops: removes, Inverse: []any{}, Notes: append([]any{head}, notes...), Targets: used,
		Keys: obj("rollback_of", bid),
		Data: &rollbackData{rollbackOf: bid, targets: used, ordOf: ordOfH, copies: copies}}, nil
}

// replicaTargetNodes loads a recorded target once: under today's write scope, with the handle the
// batch recorded (a replaced parent is refused, as every rollback does).
func replicaTargetNodes(c plan.Client, w *plan.State, targets *pyjson.Object, where string, loaded map[string]*pyjson.Object) (*pyjson.Object, error) {
	if nodes, ok := loaded[where]; ok {
		return nodes, nil
	}
	if err := w.Scope.Check(where); err != nil {
		return nil, err
	}
	h, nodes, err := plan.Handle(c, where, 1)
	if err != nil {
		return nil, err
	}
	if recorded := get(targets, where); !box.PyEqual(h, recorded) {
		return nil, plan.ToolError(where + " now has handle " + pyjson.Str(h) + " but the batch recorded " +
			pyjson.Str(recorded) + ": the component was replaced, refusing")
	}
	loaded[where] = nodes
	return nodes, nil
}

// acceptedCopyState is the verdict of a copy the station accepted under name.
func acceptedCopyState(nodes *pyjson.Object, name, typ any) string {
	n, _ := name.(string)
	node := box.Node(nodes, n)
	switch {
	case n == "" || node == nil:
		return copyGone
	case !box.PyEqual(get(node, "t"), typ):
		return copyChanged
	}
	return copyFound
}

// copyNote is one copy's verdict as a plan note.
func copyNote(e *pyjson.Object) string {
	why := map[string]string{
		copyFound:      "accepted by the station and found as added: removed",
		copyGone:       "accepted by the station, no longer there: nothing to remove",
		copyChanged:    "accepted by the station, but its name now holds another type: kept",
		copyUnverified: "this add failed and may or may not have been applied: kept, check it by hand",
		copyNotSent:    "never sent (after the failed add): kept, a component of that name is not this batch's",
		copyLaterWork:  "found as added, but a later batch wrote under it: kept, roll that batch back first",
	}[pyjson.Str(get(e, "state"))]
	return fmt.Sprintf("op %v, %s %s: %s (%s)", get(e, "op"), pyjson.Str(get(e, "target")),
		pyjson.Repr(get(e, "name")), pyjson.Str(get(e, "state")), why)
}

func strs(v []any) []string {
	out := make([]string, len(v))
	for i, s := range v {
		out[i] = pyjson.Str(s)
	}
	return out
}
