package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// checkTargetScope checks a recorded target of a rollback against today's scope. A target outside it is
// still admitted only for the rollback of an n4_create_component batch whose single inverse op removes the
// write-scope root itself: the undo of a create of the root (createScope admits it on the created path, so
// its rollback is checked on that same path, never on the parent). Only that batch kind passes laterWork,
// which refuses when later work or hand-made children sit inside the root; a copy, replicate or deploy batch
// records the same remove-from-parent op but no such check, so it stays refused (verifier of #809, P2/P2b).
func checkTargetScope(s *ord.Scope, target string, handle any, ops []any, tool any) error {
	err := s.Check(target)
	if err == nil {
		return nil
	}
	if tool != createTool || len(ops) != 1 {
		return err
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
