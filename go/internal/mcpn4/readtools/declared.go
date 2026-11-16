package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Declared properties at their default (lab retro 2026-11-15 row 1): a BOX load omits a frozen
// property still at its type default, so a fresh component reads no slots at all. A listing adds
// each property its type contract declares (the registry's loadTypes, read once per session)
// that the load left out, as the contract's default value (plan.Resolve, the same default the
// writes use); a contract that states no default gives no value, never an invented one.

// The notes of a listing whose defaults are incomplete: the property's own, and the reply's.
const (
	noDefaultNote       = "at its declared default, but the type contract gives no default value (a component default is not sent): read it in Workbench"
	contractsUnreadNote = "the station's type registry could not be read (%s): declared properties at their default are not listed"
	notDescribedNote    = "the station did not describe type %s: its declared properties at their default are not listed"
	unknownTypeNote     = "the station does not know type %s: its declared properties at their default are not listed"
)

// declared is a listing with its defaults: nodes holding the declared defaults the load left
// out, the names added under each listed path (inherited first, as declared), every added path
// (the property and the children of its default) and a note when a contract could not be read.
type declared struct {
	nodes  *pyjson.Object
	added  map[string][]string
	paths  map[string]bool
	note   string
	copied bool // nodes is this listing's own copy, not the loaded nodes
}

// withDefaults adds to nodes the declared properties at their default of the components at
// paths. A client that cannot read type contracts adds nothing and says nothing.
func (t *Tools) withDefaults(nodes *pyjson.Object, paths []string) *declared {
	d := &declared{nodes: nodes, added: map[string][]string{}, paths: map[string]bool{}}
	tl, readsTypes := t.Ctx.Session.Client.(typeLoader)
	pc, resolves := t.Ctx.Session.Client.(plan.Client)
	if !readsTypes || !resolves {
		return d
	}
	specs, seen := []string{}, map[string]bool{}
	for _, p := range paths {
		if spec := nodeType(nodes, p); spec != "" && !seen[spec] {
			seen[spec] = true
			specs = append(specs, spec)
		}
	}
	types, err := loadContracts(tl, &t.undescribed, specs)
	if err != nil {
		d.note = t.Ctx.Scrub(fmt.Sprintf(contractsUnreadNote, err.Error()))
		return d
	}
	notes := map[string]bool{}
	var order []string
	for _, p := range paths {
		names, note := declaredProps(types, nodeType(nodes, p))
		if note != "" && !notes[note] {
			notes[note] = true
			order = append(order, note)
		}
		if err := d.add(pc, p, names); err != nil {
			d.note = t.Ctx.Scrub(fmt.Sprintf(contractsUnreadNote, err.Error()))
			return &declared{nodes: nodes, added: map[string][]string{}, paths: map[string]bool{}, note: d.note}
		}
	}
	d.note = strings.Join(order, "; ")
	return d
}

// add resolves each declared name the load left out under path to its declared default. The
// lookup runs on the owner's node alone (plan.Resolve copies what it is given), and the default
// with its children joins d.nodes, copied from the loaded nodes once.
func (d *declared) add(pc plan.Client, path string, names []string) error {
	owner := pyjson.NewObject()
	owner.Set("", box.Node(d.nodes, path))
	for _, name := range names {
		full := childPath(path, name)
		if box.Node(d.nodes, full) != nil {
			continue
		}
		got, err := plan.Resolve(pc, owner, name)
		if err != nil {
			return err
		}
		if !d.copied {
			d.nodes, d.copied = copyNodes(d.nodes), true
		}
		d.added[path] = append(d.added[path], name)
		d.paths[full] = true
		for _, k := range got.Keys() {
			if k != "" {
				v, _ := got.Get(k)
				d.nodes.Set(childPath(path, k), v)
				d.paths[childPath(path, k)] = true
			}
		}
	}
	return nil
}

func copyNodes(nodes *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range nodes.Keys() {
		v, _ := nodes.Get(k)
		out.Set(k, v)
	}
	return out
}

// declaredProps is the property names spec and its supertypes declare, inherited first (a name a
// subtype declares again keeps its first place), and a note when the walk met a type the station
// did not describe or does not know. An entry {} declares nothing and ends the walk.
func declaredProps(types *pyjson.Object, spec string) (names []string, note string) {
	var chain []*pyjson.Object
	for i := 0; i < maxSupertypes && spec != ""; i++ {
		e := box.Node(types, spec)
		if e == nil {
			note = fmt.Sprintf(notDescribedNote, spec)
			break
		}
		if v, _ := e.Get("isErr"); v == true {
			note = fmt.Sprintf(unknownTypeNote, spec)
			break
		}
		chain = append(chain, e)
		spec = box.Supertype(types, spec)
	}
	seen := map[string]bool{}
	for i := len(chain) - 1; i >= 0; i-- {
		slots, _ := chain[i].Get("c")
		list, _ := slots.([]any)
		for _, s := range list {
			slot, _ := s.(*pyjson.Object)
			if slot == nil {
				continue
			}
			kind, _ := slot.Get("st")
			name, isStr := slotName(slot)
			if kind == "p" && isStr && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names, note
}

func slotName(slot *pyjson.Object) (string, bool) {
	v, _ := slot.Get("n")
	s, ok := v.(string)
	return s, ok && s != ""
}

// nodeType is the type spec of the node at path, "" when it has none.
func nodeType(nodes *pyjson.Object, path string) string {
	n := box.Node(nodes, path)
	if n == nil {
		return ""
	}
	v, _ := n.Get("t")
	s, _ := v.(string)
	return s
}

func childPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "/" + name
}
