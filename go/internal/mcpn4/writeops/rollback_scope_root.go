package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// checkTargetScope checks a recorded target of a rollback against today's scope. A target outside it is
// still admitted when every inverse op on it removes the write-scope root itself: that is the undo of a
// create of the root (createScope admits it on the created path, so its rollback is checked on that same
// path, never on the parent). Anything else on that parent stays refused.
func checkTargetScope(s *ord.Scope, target string, handle any, ops []any) error {
	err := s.Check(target)
	if err == nil {
		return nil
	}
	onTarget := 0
	for _, op := range ops {
		o, _ := op.(*pyjson.Object)
		if !box.PyEqual(get(o, "h"), handle) {
			continue
		}
		onTarget++
		if get(o, "nm") != "v" {
			return err
		}
		if root, _ := s.CheckChild(target, get(o, "n")); !root {
			return err
		}
	}
	if onTarget == 0 {
		return err
	}
	return nil
}
