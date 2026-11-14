// Package boxstation is an in-memory fake Niagara BOX station served over plain HTTP on
// 127.0.0.1: the Go port of mcp-n4-kit/tests/fake_station.py (FakeStation) for black-box tests
// that drive the real mcp-n4 binary. It serves BOX POST frames (ssession and reg channels) and
// GET /obix/about/; /file/ and /ord/ answer 404.
package boxstation

import (
	"fmt"
	"sync"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
)

// Node is one component or slot of the station tree.
type Node struct {
	Name, Type, Handle string
	Value              any // nil when the node has no value
	Children           []*Node
}

// Child is the direct child named name, nil when absent.
func (n *Node) Child(name string) *Node {
	for _, c := range n.Children {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// isComponentType is the write tools' rule (box.IsComponentType), shared with the server so the
// fake creates components exactly where the server expects them.
var isComponentType = box.IsComponentType

// defaults are the type defaults the station omits from events.
var defaults = map[string]string{"baja:Status": "0", "baja:Double": "0.0", "baja:Boolean": "false"}

// writables own a frozen proxyExt child from the moment they exist.
var writables = map[string]bool{"control:NumericWritable": true, "control:BooleanWritable": true}

// Station is the fake station. Lock mu (through Do) to read or change the tree while it serves.
type Station struct {
	User, Password string
	// ConfigPath, when set, is the config.bog a save rewrites (its mtime moves 5 s ahead).
	ConfigPath string
	// AfterAdd runs after an applied add of node, AfterInvoke after an accepted action; both run
	// with the station locked.
	AfterAdd    func(st *Station, node *Node)
	AfterInvoke func(st *Station, action string, node *Node)
	URL         string

	mu       sync.Mutex
	root     *Node
	byHandle map[string]*Node
	next     int
	events   []any
	requests int
	saves    int
}

// New is a station named name with the login user / password and the Folder component.
func New(user, password, name string) *Station {
	st := &Station{User: user, Password: password, next: 0x10, byHandle: map[string]*Node{}}
	st.root = &Node{Type: "baja:Station", Handle: "2"}
	st.byHandle["2"] = st.root
	st.root.Children = append(st.root.Children, &Node{Name: "stationName", Type: "baja:String", Value: name})
	st.Component(st.root, "Folder", "baja:Folder", "3")
	return st
}

// Root is the station root (lock with Do while the station serves).
func (st *Station) Root() *Node { return st.root }

// Do runs fn with the station locked.
func (st *Station) Do(fn func(st *Station)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	fn(st)
}

// Requests is how many BOX POSTs reached the station; Saves how many station saves it ran.
func (st *Station) Requests() (n int) { st.Do(func(*Station) { n = st.requests }); return n }

// Saves is how many station saves the station ran.
func (st *Station) Saves() (n int) { st.Do(func(*Station) { n = st.saves }); return n }

// Has reports whether the root holds a child named name.
func (st *Station) Has(name string) (ok bool) {
	st.Do(func(*Station) { ok = st.root.Child(name) != nil })
	return ok
}

// Component adds a component under parent with a fresh handle unless one is given.
func (st *Station) Component(parent *Node, name, typ, handle string) *Node {
	if handle == "" {
		handle = fmt.Sprintf("%x", st.next)
		st.next++
	}
	n := &Node{Name: name, Type: typ, Handle: handle}
	parent.Children = append(parent.Children, n)
	st.byHandle[handle] = n
	return n
}

func (st *Station) unique(parent *Node, name string) string {
	candidate := name
	for i := 1; parent.Child(candidate) != nil; i++ {
		candidate = fmt.Sprintf("%s%d", name, i)
	}
	return candidate
}

// fill sets node from a BOX body: type, value, and its children (nested components get handles).
func (st *Station) fill(n *Node, b map[string]any) {
	if t, ok := b["t"].(string); ok {
		n.Type = t
	}
	n.Value = b["v"]
	n.Children = nil
	subs, _ := b["s"].([]any)
	for _, raw := range subs {
		sub, _ := raw.(map[string]any)
		name, _ := sub["n"].(string)
		typ, _ := sub["t"].(string)
		var child *Node
		if isComponentType(sub["t"]) {
			child = st.Component(n, name, typ, "")
		} else {
			child = &Node{Name: name, Type: typ}
			n.Children = append(n.Children, child)
		}
		st.fill(child, sub)
	}
}

// serialize is a node as a BOX body to depth, omitting slots at their type default.
func serialize(n *Node, depth int, isRoot bool) map[string]any {
	out := map[string]any{"nm": "p", "t": n.Type}
	if !isRoot {
		out["n"] = n.Name
	}
	if n.Handle != "" {
		out["h"] = n.Handle
	}
	if n.Value != nil {
		out["v"] = n.Value
	}
	if depth > 0 {
		var kids []any
		for _, c := range n.Children {
			if d, ok := defaults[c.Type]; ok && c.Value == d {
				continue
			}
			kids = append(kids, serialize(c, depth-1, false))
		}
		if len(kids) > 0 {
			out["s"] = kids
		}
	}
	return out
}
