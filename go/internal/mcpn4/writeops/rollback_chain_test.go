package writeops

// n4_rollback chain=true (issue #971): what its plan says about undoing it (item 1) and the rollback of
// an ancestor create after it (item 2). The journal is lwJournal (rollback_later_test.go): A created
// Outer, C created Box under it, B1 created Calc under Box; R is the chain rollback of C.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// lwStation answers a load of an ORD with its fixed nodes, whatever the depth; it never sends.
type lwStation map[string]*pyjson.Object

func (s lwStation) LoadTree(o string, _ int, _ box.LoadOptions) (*pyjson.Object, error) {
	if n, ok := s[o]; ok {
		return n, nil
	}
	return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + o}
}
func (lwStation) SSC(string, any) (any, error) { panic("a plan never sends") }
func (lwStation) Sync(any) (any, error)        { panic("a plan never sends") }
func (lwStation) ClearHandles()                {}

func lwNode(h, t string) *pyjson.Object { return obj("nm", "p", "h", h, "t", t) }

// lwChainStation is the station after A, C and B1 (Calc still under Box), or, with gone, after the chain
// rollback of C removed Box.
func lwChainStation(gone bool) lwStation {
	outer := obj("", lwNode("4", "baja:Folder"))
	if !gone {
		outer.Set("Box", lwNode("5", "baja:Folder"))
		outer.Set("Box/Calc", lwNode("6", "kitControl:NumericConst"))
	}
	return lwStation{"station:|slot:/Folder": obj("", lwNode("3", "baja:Folder"), "Outer", lwNode("4", "baja:Folder")),
		lwOuter: outer}
}

func lwPlan(t *testing.T, c lwStation, w *plan.State, id string, chain bool) (*plan.Planned, error) {
	t.Helper()
	scope, err := ord.NewScope([]string{"station:|slot:/Folder"})
	if err != nil {
		t.Fatal(err)
	}
	w.Scope = scope
	return rollbackPlan(c, obj("batch_id", id, "chain", chain), w, &session.Session{StationName: "S"})
}

// lwB1 is B1 of lwJournal with the op that added Calc, which the chain covers.
func lwB1(views []*pyjson.Object) {
	views[2].Set("ops", []any{obj("nm", "a", "h", "5", "n", "Calc")})
}

// The plan of a chain says what it removes and that the rollback has no automatic inverse: the created
// component and the later work under it are removed without a snapshot, so the chain cannot be undone
// (decision for #971 item 1: the plan note says so, no snapshot is taken).
func TestChainRollbackPlanHasNoInverseAndSaysSo(t *testing.T) {
	w, views := lwJournal()
	lwB1(views)
	w.Journal = viewsOnly{views[:3]} // A, C, B1: the chain is not run yet
	p, err := lwPlan(t, lwChainStation(false), w, "C", true)
	if err != nil {
		t.Fatal(err)
	}
	notes := esNotes(p)
	for _, want := range []string{"chain: removes the created " + lwBox + " with the later batch(es) under it, newest first: B1 (n4_create_component)",
		"the rollback itself has no automatic inverse for slot restores or removals"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes lack %q: %s", want, notes)
		}
	}
	if len(p.Inverse) != 0 {
		t.Errorf("a chain plan has an inverse: %s", pyjson.Repr(list(p.Inverse)))
	}
	if got, err := rollbackInverse(p, []any{nil}); err != nil || len(got) != 0 {
		t.Errorf("the executed chain records inverse %v, %v; want none", got, err)
	}
}

func esNotes(p *plan.Planned) string {
	var out []string
	for _, n := range p.Notes {
		out = append(out, pyjson.Str(n))
	}
	return strings.Join(out, "\n")
}

// After the chain the later batches stay "not undone" in the journal, yet a plain rollback of the ancestor
// create A plans: they are gone with Box. Before the chain it refuses on them (#971 item 2).
func TestRollbackOfAnAncestorCreateAfterAChainDoesNotRefuseOnVanishedBatches(t *testing.T) {
	w, views := lwJournal()
	lwB1(views)
	p, err := lwPlan(t, lwChainStation(true), w, "A", false)
	if err != nil {
		t.Fatalf("second rollback after the chain: %v", err)
	}
	if got := pyjson.Repr(list(p.Ops)); !strings.Contains(got, "'n': 'Outer'") {
		t.Errorf("ops = %s, want the remove of Outer", got)
	}
	w.Journal = viewsOnly{views[:4]} // the chain never ran: C, B1 and the link are later work
	_, err = lwPlan(t, lwChainStation(false), w, "A", false)
	if err == nil || !strings.Contains(err.Error(), "later batch(es) wrote under it: B2 (n4_create_link), B1 (n4_create_component), C (n4_create_component)") {
		t.Errorf("without the chain: %v", err)
	}
}
