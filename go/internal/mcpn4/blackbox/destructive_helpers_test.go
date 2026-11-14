package blackbox

// The rig of the destructive-tool tests (port of test_tools_destructive.DestructiveCase): a
// writes-allowed server connected with a verified identity, the station arranged through BOX
// ops, and the fault hook of destructive_faults_test.go installed.

import (
	"fmt"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// Limits and wording of the server's write engine; the tests find each one in a real plan or
// refusal, so a drift fails them.
const (
	snapshotDepth            = 3
	linkScanDepth            = 6
	reasonJournalUnreadable  = "journal line unreadable"
	destructiveStationRootOr = "station:|slot:/"
)

// dRig is a Rig with the fault hook.
type dRig struct {
	*Rig
	f *faults
}

// destructiveRig starts the rig (configure, when not nil, edits the options after
// Rig.WriteOptions) and connects with expected_station.
func destructiveRig(t *testing.T, configure func(r *Rig, o *Options)) *dRig {
	d := &dRig{f: &faults{}}
	d.Rig = NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		d.f.install(r.Station)
		if configure != nil {
			configure(r, o)
		}
	})
	d.ConnectVerified()
	return d
}

// restart is start_server(**kw) then connect_verified: a new server with edited options.
func (d *dRig) restart(configure func(o *Options)) {
	d.T.Helper()
	d.Restart(configure)
	d.ConnectVerified()
}

// sync sends one syncTo op as box_client.sync does and returns its reply or its BOX error.
func (d *dRig) sync(op map[string]any) (any, error) {
	ans := d.Station.Answer(map[string]any{"v": "2.3", "p": "box", "n": 1, "m": []any{
		map[string]any{"c": "ssession", "k": "callssc", "r": 0, "t": "rq",
			"b": map[string]any{"sck": "syncTo", "scarg": map[string]any{"ops": []any{op}}}}}})
	return boxReply(ans)
}

// loadSlots asks the station to load ord, as box_client.load_tree does; only its error is kept.
func (d *dRig) loadSlots(ord string) error {
	ans := d.Station.Answer(map[string]any{"v": "2.3", "p": "box", "n": 1, "m": []any{
		map[string]any{"c": "ssession", "k": "callssc", "r": 0, "t": "rq",
			"b": map[string]any{"sck": "loadSlots", "scarg": map[string]any{"o": ord, "d": 1.0}}}}})
	_, err := boxReply(ans)
	return err
}

func boxReply(ans map[string]any) (any, error) {
	msg := objects(ans["m"])[0]
	body, _ := msg["b"].(map[string]any)
	if msg["t"] == "e" {
		return nil, fmt.Errorf("%v", body["m"])
	}
	return msg["b"], nil
}

// addBody adds a component with a whole BOX body under the component with handle parent and
// returns the name the station gave it (box_client.add_component).
func (d *dRig) addBody(parent, name string, body map[string]any) string {
	d.T.Helper()
	res, err := d.sync(map[string]any{"nm": "a", "h": parent, "n": name, "b": body})
	if err != nil {
		d.T.Fatal(err)
	}
	return objects(res)[0]["nn"].(string)
}

// addOut adds a component of typ under the Folder, with an out of value when out is not nil
// (WriteTestCase.add); it returns the name and handle.
func (d *dRig) addOut(name, typ string, out *float64) (string, string) {
	d.T.Helper()
	c := d.Add("3", name, typ)
	if out != nil {
		d.Set(c.Handle, "out", StatusNumeric(*out, "0"))
	}
	return c.Name, c.Handle
}

func num(v float64) *float64 { return &v }

// wsAnnotation is box_client.ws_annotation.
func wsAnnotation(x, y, w int) map[string]any {
	return map[string]any{"nm": "p", "n": "wsAnnotation", "t": "baja:WsAnnotation", "v": fmt.Sprintf("%d,%d,%d", x, y, w)}
}

// group is DestructiveCase.group: a baja:Folder under the Folder with a wsAnnotation, two
// NumericConsts (Src with out 4.5, Tgt) and a link Src.out -> Tgt.in10.
func (d *dRig) group(name string) (nn, gh string) {
	d.T.Helper()
	nn = d.addBody("3", name, map[string]any{"nm": "p", "t": "baja:Folder", "s": []any{wsAnnotation(1, 2, 3)}})
	gh = d.node(nn).Handle
	src := d.Add(gh, "Src", "")
	tgt := d.Add(gh, "Tgt", "")
	d.Set(src.Handle, "out", StatusNumeric(4.5, "0"))
	d.Link(src, "out", tgt, "in10")
	return nn, gh
}

// node is the node at path under the Folder, nil when absent; its fields are read while the
// server is idle (between calls).
func (d *dRig) node(path ...string) *boxstation.Node { return d.nodeFrom("3", path...) }

// nodeFrom is the node at path under the component with handle h, nil when absent.
func (d *dRig) nodeFrom(h string, path ...string) (n *boxstation.Node) {
	d.Station.Do(func(st *boxstation.Station) {
		n = st.ByHandle(h)
		for _, p := range path {
			if n = n.Child(p); n == nil {
				return
			}
		}
	})
	return n
}

// value is the value of the slot at path under the component with handle h.
func (d *dRig) value(h string, path ...string) (v any) {
	d.Station.Do(func(st *boxstation.Station) {
		n := st.ByHandle(h)
		for _, p := range path {
			if n = n.Child(p); n == nil {
				return
			}
		}
		v = n.Value
	})
	return v
}

// childValues maps the children of the node at path (under the Folder) to their values.
func (d *dRig) childValues(path ...string) map[string]any {
	out := map[string]any{}
	n := d.node(path...)
	if n == nil {
		return out
	}
	d.Station.Do(func(*boxstation.Station) {
		for _, c := range n.Children {
			out[c.Name] = c.Value
		}
	})
	return out
}

// childNames are the names of a node's children, read with the station locked.
func (d *dRig) childNames(n *boxstation.Node) []string {
	out := []string{}
	d.Station.Do(func(*boxstation.Station) {
		for _, c := range n.Children {
			out = append(out, c.Name)
		}
	})
	return out
}

// children is WriteTestCase.children: the Folder's children, never nil.
func (d *dRig) children() []string { return d.childNames(d.node()) }

// removeChild drops the child named name of n (the model edit a patched _sync made).
func removeChild(n *boxstation.Node, name string) {
	if n == nil {
		return
	}
	kept := n.Children[:0]
	for _, c := range n.Children {
		if c.Name != name {
			kept = append(kept, c)
		}
	}
	n.Children = kept
}

// afterAdd runs fn after every applied add, with the station locked (a patched _sync).
func (d *dRig) afterAdd(fn func(st *boxstation.Station, n *boxstation.Node)) {
	d.Station.Do(func(st *boxstation.Station) { st.AfterAdd = fn })
}

// removeGroup is removed_group: a group removed by a confirmed n4_remove_component.
func (d *dRig) removeGroup() (string, map[string]any) {
	d.T.Helper()
	nn, _ := d.group("Grp")
	return nn, d.Srv.RunWrite("n4_remove_component", Args{"parent_ord": Folder, "name": nn})
}

// rollback runs a confirmed n4_rollback of id.
func (d *dRig) rollback(id any) map[string]any {
	d.T.Helper()
	return d.Srv.RunWrite("n4_rollback", Args{"batch_id": id})
}

func removeArgs(nn string) Args { return Args{"parent_ord": Folder, "name": nn} }
