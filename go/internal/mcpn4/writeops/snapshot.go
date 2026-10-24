package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

const (
	// SnapshotDepth is SNAPSHOT_DEPTH: how many levels below the component the snapshot keeps.
	SnapshotDepth = 3
	// TruncatedPathsShown is TRUNCATED_PATHS_SHOWN: the paths a plan lists below the cut.
	TruncatedPathsShown = 20
)

// runtimeOutputSlots is RUNTIME_OUTPUT_SLOTS: Status* outputs a re-create body never restores.
var runtimeOutputSlots = map[string]bool{"out": true}

// linkRec is a link the snapshot met: the path of the component that holds it, its slots and
// the configuration-only value of the input it drives (prior), when that input was captured.
type linkRec struct {
	targetPath                        string
	sourceOrd, sourceSlot, targetSlot any
	prior                             *pyjson.Object
}

// snapshot is _snapshot's shared state: the links met and handle -> path (keyed as a dict).
type snapshot struct {
	links   []*linkRec
	handles map[string]string
}

// kidsOf is node.get("s", []) of a node _prune cut to depth levels of children.
func kidsOf(node *pyjson.Object, depth int) []*pyjson.Object {
	if depth <= 0 {
		return nil
	}
	raw, _ := get(node, "s").([]any) // LoadTree checked: a list of named objects
	out := make([]*pyjson.Object, 0, len(raw))
	for _, k := range raw {
		out = append(out, k.(*pyjson.Object))
	}
	return out
}

func isLink(t any) bool { return t == "baja:Link" || t == "baja:ConversionLink" }

// statusType is str(t).startswith("baja:Status"): only a string can start so.
func statusType(t any) bool { s, _ := t.(string); return strings.HasPrefix(s, "baja:Status") }

func name(node *pyjson.Object) string { n, _ := get(node, "n").(string); return n }

// take is _snapshot(_prune(node, depth), path, links, handles): type, value and children as
// data, no handles. Links are recorded apart; a link-driven input is kept on its link as prior.
func (s *snapshot) take(node *pyjson.Object, path string, depth int) (*pyjson.Object, error) {
	if h := get(node, "h"); pyjson.Truthy(h) {
		key, err := pyjson.DictKey(h)
		if err != nil {
			return nil, unhashable(h)
		}
		s.handles[key] = path
	}
	out := obj("nm", "p")
	if t := get(node, "t"); t != nil { // frozen slot children carry no type: never "t": null
		out.Set("t", t)
	}
	if n, ok := node.Get("n"); path != "" && ok {
		out.Set("n", n)
	}
	if v := get(node, "v"); v != nil {
		out.Set("v", v)
	}
	children := kidsOf(node, depth)
	driven := map[string]bool{}
	for _, c := range children {
		if !isLink(get(c, "t")) {
			continue
		}
		for _, k := range kidsOf(c, depth-1) {
			if name(k) == "targetSlotName" {
				key, err := pyjson.DictKey(get(k, "v"))
				if err != nil {
					t := plan.TypeName(get(k, "v"))
					return nil, &plan.PyError{Type: "TypeError", Msg: "cannot use '" + t + "' as a set element (unhashable type: '" + t + "')"}
				}
				driven[key] = true
			}
		}
	}
	kids, first, prior := []any{}, len(s.links), map[string]*pyjson.Object{}
	for _, child := range children {
		n := name(child)
		childPath := n
		if path != "" {
			childPath = path + "/" + n
		}
		if isLink(get(child, "t")) {
			slots := pyjson.NewObject()
			for _, k := range kidsOf(child, depth-1) {
				slots.Set(name(k), get(k, "v"))
			}
			s.links = append(s.links, &linkRec{targetPath: path, sourceOrd: get(slots, "sourceOrd"),
				sourceSlot: get(slots, "sourceSlotName"), targetSlot: get(slots, "targetSlotName")})
			continue
		}
		if runtimeOutputSlots[n] && statusType(get(child, "t")) && !driven["s"+n] {
			continue
		}
		kid, err := s.take(child, childPath, depth-1)
		if err != nil {
			return nil, err
		}
		if _, has := kid.Get("v"); has && (get(kid, "t") == "baja:Status" || n == "status" && statusType(get(node, "t"))) {
			// keep the configured status bits (null, disabled...), drop runtime facets after ';'
			kid.Set("v", strings.SplitN(pyjson.Str(get(kid, "v")), ";", 2)[0])
		}
		if driven["s"+n] { // kept on its link, never in the re-create body
			prior[n] = kid
			continue
		}
		kids = append(kids, kid)
	}
	for _, l := range s.links[first:] {
		if l.targetPath != path {
			continue
		}
		if t, ok := l.targetSlot.(string); ok && prior[t] != nil {
			l.prior = prior[t]
		}
	}
	if len(kids) > 0 {
		out.Set("s", kids)
	}
	return out, nil
}
