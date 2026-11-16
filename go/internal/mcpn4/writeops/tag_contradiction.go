package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// contradiction refuses a request that both adds and removes one tag, tag group or relation:
// the batch would end in whichever op ran last, and its read-back could only mismatch.
func (r *tagRequest) contradiction() error {
	removed := map[tags.ID]bool{}
	for _, id := range r.removes {
		removed[id] = true
	}
	for _, a := range r.adds {
		if removed[a.id] {
			return both(a.id.String())
		}
	}
	for _, pair := range [][2][]tagRel{{r.addGroups, r.rmGroups}, {r.addRels, r.rmRels}} {
		for _, a := range pair[0] {
			for _, d := range pair[1] {
				if a.same(d) {
					return both(a.label)
				}
			}
		}
	}
	return nil
}

func both(what string) error {
	return plan.ToolError(what + " is both added and removed in one request: refusing (send two batches)")
}
