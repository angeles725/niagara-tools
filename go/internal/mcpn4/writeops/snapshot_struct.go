package writeops

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// structOracle classifies by the contracts a snapshot read (B1200-G3); nil without contracts, which
// keeps the depth count of every slot as before.
func structOracle(ct *plan.Contracts) *plan.RegistryOracle {
	if ct == nil {
		return nil
	}
	return plan.NewRegistryOracle(ct)
}

// below is the depth left for the children of child, a child of a node taken at depth: one level
// less, except for a struct (alarm:LimitEnable, a baja:Struct by the station registry), which is
// part of the body of the component holding it, so its slots are captured at the same depth as
// the struct itself. SnapshotDepth counts component levels.
func (s *snapshot) below(child *pyjson.Object, depth int) int {
	if s.oracle.IsStruct(get(child, "t")) {
		return depth
	}
	return depth - 1
}

// visit records that take reached path.
func (s *snapshot) visit(path string) {
	if s.taken == nil {
		s.taken = map[string]bool{}
	}
	s.taken[path] = true
}

// truncatedPaths are the loaded slots at SnapshotDepth or deeper that take did not reach, in code
// point order (WTF-8 bytes sort as code points), and the nodes it did reach at the load cut that may
// hold children the load never read (cutShort); a link's own slots are read into its record when
// the plan has a scope (wholeLinks) and are not listed.
func (s *snapshot) truncatedPaths(nodes *pyjson.Object, wholeLinks bool) []string {
	var out []string
	for _, p := range nodes.Keys() {
		if wholeLinks && linkSlot(nodes, p) {
			continue
		}
		if p != "" && strings.Count(p, "/") >= SnapshotDepth && (!s.taken[p] || s.cutShort(box.Node(nodes, p))) {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// cutShort says whether node, reached by take on the last level the load holds (a struct kept at its
// parent's depth, below), has a type that can hold slots: its children were not loaded, so they are
// not in the snapshot and must be reported, never taken for complete. A simple (a boolean) has none.
func (s *snapshot) cutShort(node *pyjson.Object) bool {
	t := get(node, "t")
	return s.oracle.IsStruct(t) || plan.ClassifyComponent(s.oracle, t)
}
