package writeops

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// scanRoot is a plan.Client whose every load answers one tree (the scan root's); it never sends.
type scanRoot struct{ nodes *pyjson.Object }

func (s scanRoot) LoadTree(string, int, box.LoadOptions) (*pyjson.Object, error) { return s.nodes, nil }
func (scanRoot) SSC(string, any) (any, error)                                    { panic("a plan never sends") }
func (scanRoot) Sync(any) (any, error)                                           { panic("a plan never sends") }
func (scanRoot) ClearHandles()                                                   {}

// A link held by a component more than 3 levels inside the removed subtree has a holder handle
// the depth-3 snapshot never recorded; its target path is still inside the subtree, so it is not
// an outgoing link (review R3-deep-internal-links-flagged). The fake station cannot show it at the
// current depths (the link's slots sit one level below the scan), a station that sends them can.
func TestALinkHeldDeepInsideTheRemovedSubtreeIsNotOutgoing(t *testing.T) {
	tree, err := pyjson.Unmarshal([]byte(`{"t": "baja:Folder", "h": "3", "s": [
		{"n": "Grp", "t": "baja:Folder", "h": "10", "s": [
			{"n": "Src", "t": "kitControl:NumericConst", "h": "20"},
			{"n": "L1", "t": "baja:Folder", "h": "11", "s": [{"n": "L2", "t": "baja:Folder", "h": "12", "s": [
				{"n": "L3", "t": "baja:Folder", "h": "13", "s": [{"n": "Deep", "t": "kitControl:NumericConst", "h": "14", "s": [
					{"n": "Link", "t": "baja:Link", "s": [
						{"n": "sourceOrd", "v": "h:20"}, {"n": "sourceSlotName", "v": "out"},
						{"n": "targetSlotName", "v": "in10"}]}]}]}]}]}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	nodes := pyjson.NewObject()
	nodes.Set("", tree)
	scope, err := ord.NewScope([]string{"station:|slot:/Folder"})
	if err != nil {
		t.Fatal(err)
	}
	args := obj("parent_ord", "station:|slot:/Folder", "name", "Grp")
	handles := map[string]string{"s10": "", "s20": "Src", "s11": "L1", "s12": "L1/L2", "s13": "L1/L2/L3"}
	keys := pyjson.NewObject()
	_, inverse, err := linkScan(scanRoot{nodes}, args, scope, nodes, handles, "", keys, nil, nil)
	if err != nil {
		t.Fatalf("an internal link refused the remove: %v", err)
	}
	if _, ok := keys.Get("outgoing_links_broken"); ok || len(inverse) != 0 {
		t.Errorf("outgoing_links_broken %v, inverse %v", get(keys, "outgoing_links_broken"), inverse)
	}
}
