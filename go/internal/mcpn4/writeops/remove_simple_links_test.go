package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// echoStation is a station that honours the load depth (one level per slot, as the fake BOX
// station does) and answers a simple or struct slot's ORD with the component that holds it, as
// a 4.15 station does. Folder (h 3) holds P (h 6) and Q (h 7); Q.Link reads P.x, Q.Link1 reads
// P.out; P.Link writes P.dyn from Q.out.
type echoStation struct{ trees map[string]*pyjson.Object }

const (
	esFolder = "station:|slot:/Folder"
	esP      = esFolder + "/P"
)

func esNode(n, t string, kv ...any) *pyjson.Object {
	o := obj("nm", "p", "n", n, "t", t)
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func esLink(n, src, ss, ts string) *pyjson.Object {
	return esNode(n, "baja:Link", "s", []any{esNode("sourceOrd", "baja:Ord", "v", "h:"+src),
		esNode("sourceSlotName", "baja:String", "v", ss), esNode("targetSlotName", "baja:String", "v", ts)})
}

func newEchoStation() *echoStation {
	p := obj("nm", "p", "h", "6", "t", "control:NumericWritable", "s", []any{
		esNode("tst$3acap", "baja:Double", "v", "4.5", "f", "m"),
		esNode("x", "baja:Double", "v", "2.0"),
		esNode("dyn", "baja:Double", "v", "3.0"),
		esLink("Link", "7", "out", "dyn"),
		esNode("sn", "baja:StatusNumeric", "s", []any{esNode("value", "baja:Double", "v", "7.0")}),
		obj("nm", "p", "n", "Grp", "h", "10", "t", "baja:Folder", "f", "h"),
	})
	pNamed := obj("nm", "p", "n", "P", "h", "6", "t", "control:NumericWritable")
	pNamed.Set("s", get(p, "s"))
	q := obj("nm", "p", "n", "Q", "h", "7", "t", "kitControl:Add", "s", []any{
		esLink("Link", "6", "x", "in10"), esLink("Link1", "6", "out", "in11")})
	folder := obj("nm", "p", "h", "3", "t", "baja:Folder", "s", []any{pNamed, q})
	grp := obj("nm", "p", "h", "10", "t", "baja:Folder")
	return &echoStation{trees: map[string]*pyjson.Object{esFolder: folder, esP: p, esP + "/Grp": grp}}
}

func esFlatten(n *pyjson.Object, path string, depth int, out *pyjson.Object) {
	node := pyjson.NewObject()
	for _, k := range n.Keys() {
		if k != "s" && !(path == "" && k == "n") {
			v, _ := n.Get(k)
			node.Set(k, v)
		}
	}
	out.Set(path, node)
	if depth <= 0 {
		return
	}
	kids, _ := get(n, "s").([]any)
	var shown []any
	for _, k := range kids {
		ko := k.(*pyjson.Object)
		p := get(ko, "n").(string)
		if path != "" {
			p = path + "/" + p
		}
		esFlatten(ko, p, depth-1, out)
		shown = append(shown, ko)
	}
	if len(shown) > 0 {
		node.Set("s", shown)
	}
}

func (s *echoStation) LoadTree(o string, depth int, _ box.LoadOptions) (*pyjson.Object, error) {
	t, ok := s.trees[o]
	if !ok && strings.HasPrefix(o, esP+"/") {
		t = s.trees[esP] // a simple slot's ORD answers its component
	}
	if t == nil {
		return nil, &box.Error{Kind: box.KindBox, Msg: "no such component: " + o}
	}
	out := pyjson.NewObject()
	esFlatten(t, "", depth, out)
	return out, nil
}
func (s *echoStation) SSC(string, any) (any, error) { panic("a plan never sends") }
func (s *echoStation) Sync(any) (any, error)        { panic("a plan never sends") }
func (s *echoStation) ClearHandles()                {}

func esPlan(t *testing.T, name string, extra ...any) (*plan.Planned, error) {
	t.Helper()
	scope, err := ord.NewScope([]string{esFolder})
	if err != nil {
		t.Fatal(err)
	}
	args := obj("parent_ord", esP, "name", name)
	for i := 0; i < len(extra); i += 2 {
		args.Set(extra[i].(string), extra[i+1])
	}
	return writeops.RemoveComponent.PlanWith(newEchoStation(), args, &plan.State{Scope: scope}, nil)
}

func esJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Removing a simple slot that is the source of a link into another component breaks that link:
// the plan must refuse until acknowledge_links is given, as it does for a component.
func TestRemoveSimpleSlotThatSourcesALinkIsRefusedWithoutAcknowledge(t *testing.T) {
	_, err := esPlan(t, "x")
	if err == nil {
		t.Fatal("removing P.x (read by Q.in10) planned without acknowledge_links")
	}
	if msg := err.Error(); !strings.Contains(msg, "in10 <- x") || !strings.Contains(msg, "acknowledge_links") {
		t.Fatalf("refusal = %s", msg)
	}
}

// Acknowledged, the inverse records the link the slot sources and none of the parent's other links.
func TestRemoveSimpleSlotAcknowledgedRecordsOnlyItsOwnOutgoingLink(t *testing.T) {
	p, err := esPlan(t, "x", "acknowledge_links", true)
	if err != nil {
		t.Fatal(err)
	}
	inv := esJSON(t, p.Inverse)
	if !strings.Contains(inv, `"target_slot": "in10", "source_path": "", "source_slot": "x"`) {
		t.Errorf("the link Q.in10 <- P.x is not in the inverse: %s", inv)
	}
	if strings.Contains(inv, `"source_slot": "out"`) {
		t.Errorf("the inverse re-creates a link the remove does not break (P.out -> Q.in11): %s", inv)
	}
}

// A simple slot that sources no link is not refused for the links its component's other slots source.
func TestRemoveTagIsNotRefusedForTheComponentsOtherLinks(t *testing.T) {
	p, err := esPlan(t, "tst$3acap")
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"nm": "a", "h": "6", "n": "tst$3acap", "b": {"nm": "p", "t": "baja:Double", "v": "4.5"}, "f": "m"}]`
	if got := esJSON(t, p.Inverse); got != want {
		t.Fatalf("inverse\n got %s\nwant %s", got, want)
	}
	if got := esJSON(t, p.Ops); got != `[{"nm": "v", "h": "6", "n": "tst$3acap"}]` {
		t.Fatalf("ops %s", got)
	}
}

// A simple slot written by a link loses that input: the inverse records it (unlinked_input) so
// the rollback reports it, as it does for a component's linked input.
func TestRemoveSimpleSlotLinkedIntoRecordsTheUnlinkedInput(t *testing.T) {
	p, err := esPlan(t, "dyn")
	if err != nil {
		t.Fatal(err)
	}
	if inv := esJSON(t, p.Inverse); !strings.Contains(inv, `"target_slot": "dyn"`) {
		t.Errorf("the link P.dyn <- Q.out is not recorded in the inverse: %s", inv)
	}
}

// A struct slot's fields come back on rollback even though the parent was loaded at depth 1.
func TestRemoveStructSlotKeepsItsFieldsWhateverTheParentLoadDepth(t *testing.T) {
	p, err := esPlan(t, "sn")
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"nm": "a", "h": "6", "n": "sn", "b": {"nm": "p", "t": "baja:StatusNumeric", "s": [` +
		`{"nm": "p", "t": "baja:Double", "n": "value", "v": "7.0"}]}}]`
	if got := esJSON(t, p.Inverse); got != want {
		t.Fatalf("inverse\n got %s\nwant %s", got, want)
	}
}

// A component child is planned as before: its own load, and no slot flags on the re-add.
func TestRemoveComponentChildKeepsTheComponentPlan(t *testing.T) {
	p, err := esPlan(t, "Grp")
	if err != nil {
		t.Fatal(err)
	}
	if got := esJSON(t, p.Inverse); got != `[{"nm": "a", "h": "6", "n": "Grp", "b": {"nm": "p", "t": "baja:Folder"}}]` {
		t.Fatalf("inverse %s", got)
	}
}
