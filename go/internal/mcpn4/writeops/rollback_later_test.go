package writeops

// The later work of a create's rollback after a chain=true rollback (issue #971 item 2): the batches
// the chain removed with their created component are gone, so a plain rollback of an ancestor create
// no longer refuses on them.

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// viewsOnly is a journal that holds the given merged views (it also answers what n4_rollback reads).
type viewsOnly struct{ views []*pyjson.Object }

func (v viewsOnly) Views() ([]*pyjson.Object, error) { return v.views, nil }
func (viewsOnly) Append(*pyjson.Object) error        { return nil }
func (viewsOnly) CheckPrivate() error                { return nil }

const (
	lwOuter = "station:|slot:/Folder/Outer"
	lwBox   = lwOuter + "/Box"
)

// lwBatch is one journaled batch view on station S: its targets (ORD -> handle) and inverse.
func lwBatch(id, tool string, targets *pyjson.Object, inverse ...any) *pyjson.Object {
	return obj("batch_id", id, "tool", tool, "station_name", "S", "state", "completed", "verdict", "verified",
		"targets", targets, "inverse", inverse)
}

func lwRemove(h, n string) *pyjson.Object { return obj("nm", "v", "h", h, "n", n) }

// lwRollback is the completed rollback batch of of, with the given verdict.
func lwRollback(id, of, verdict string, targets *pyjson.Object) *pyjson.Object {
	v := lwBatch(id, "n4_rollback", targets)
	v.Set("rollback_of", of)
	v.Set("verdict", verdict)
	return v
}

// lwJournal is the journal of a chain: A created Outer, C created Box under it, B1 created Calc and B2 a link
// under Box, and the chain rollback R of C removed Box with B1 and B2.
func lwJournal(extra ...*pyjson.Object) (*plan.State, []*pyjson.Object) {
	views := []*pyjson.Object{
		lwBatch("A", createTool, obj("station:|slot:/Folder", "3"), lwRemove("3", "Outer")),
		lwBatch("C", createTool, obj(lwOuter, "4"), lwRemove("4", "Box")),
		lwBatch("B1", createTool, obj(lwBox, "5"), lwRemove("5", "Calc")),
		lwBatch("B2", "n4_create_link", obj(lwBox, "5")),
		lwRollback("R", "C", "verified", obj(lwOuter, "4")),
	}
	views = append(views, extra...)
	return &plan.State{Journal: viewsOnly{views}}, views
}

func laterOf(t *testing.T, w *plan.State, views []*pyjson.Object) []string {
	t.Helper()
	later, err := laterBatches(w, views[0], "A", lwOuter)
	if err != nil {
		t.Fatal(err)
	}
	return later
}

func TestLaterBatchesSkipWhatAChainRollbackRemovedWithItsCreate(t *testing.T) {
	w, views := lwJournal()
	if got := laterOf(t, w, views); len(got) != 0 {
		t.Errorf("later work after the chain = %q, want none: Box, Calc and the link are gone", got)
	}
}

// Work journaled after the chain, under the same path, is later work again.
func TestLaterBatchesKeepWorkJournaledAfterTheChain(t *testing.T) {
	w, views := lwJournal(lwBatch("B3", createTool, obj(lwBox, "9"), lwRemove("9", "Calc")))
	if got, want := laterOf(t, w, views), []string{"B3 (n4_create_component)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("later work = %q, want %q", got, want)
	}
}

// A rollback that failed removed nothing: its create and everything under it stay later work.
func TestLaterBatchesKeepWhatAFailedChainRollbackLeft(t *testing.T) {
	w, views := lwJournal()
	views[4].Set("verdict", "failed")
	want := []string{"B2 (n4_create_link)", "B1 (n4_create_component)", "C (n4_create_component)"}
	if got := laterOf(t, w, views); !reflect.DeepEqual(got, want) {
		t.Errorf("later work = %q, want %q", got, want)
	}
}

// Without the rollback the whole chain is later work, newest first (the refusal as before).
func TestLaterBatchesWithoutTheChainRollbackListTheWholeChain(t *testing.T) {
	w, views := lwJournal()
	w.Journal = viewsOnly{views[:4]}
	want := []string{"B2 (n4_create_link)", "B1 (n4_create_component)", "C (n4_create_component)"}
	if got := laterOf(t, w, views[:4]); !reflect.DeepEqual(got, want) {
		t.Errorf("later work = %q, want %q", got, want)
	}
}

// A batch that also wrote outside the removed component was not removed with it.
func TestLaterBatchesKeepABatchThatAlsoWroteOutsideTheChainedCreate(t *testing.T) {
	w, views := lwJournal()
	views[3].Set("targets", obj(lwBox, "5", "station:|slot:/Folder/Outer/Other", "6"))
	if got, want := laterOf(t, w, views), []string{"B2 (n4_create_link)"}; !reflect.DeepEqual(got, want) {
		t.Errorf("later work = %q, want %q", got, want)
	}
}

// The rollback of a batch that is no create takes nothing else with it: only that batch is undone.
func TestLaterBatchesMarkNothingFromARollbackOfANonCreate(t *testing.T) {
	w, views := lwJournal()
	views[4].Set("rollback_of", "B2") // B2 is a link, no create
	want := []string{"B1 (n4_create_component)", "C (n4_create_component)"}
	if got := laterOf(t, w, views); !reflect.DeepEqual(got, want) {
		t.Errorf("later work = %q, want %q", got, want)
	}
}

// RollbackTarget is the view of batch id; EarlierRollbacks refuses nothing.
func (v viewsOnly) RollbackTarget(id any) (*pyjson.Object, error) {
	for _, view := range v.views {
		if box.PyEqual(get(view, "batch_id"), id) {
			return view, nil
		}
	}
	return nil, plan.ToolError("no such batch")
}

func (viewsOnly) EarlierRollbacks(string) error { return nil }
