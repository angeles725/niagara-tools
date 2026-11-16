package writeops

// The write-scope rules of the tools whose scope is more than "every ORD under a prefix" (retro
// 2026-11-13 M5 and M10).

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// createScope admits a create whose parent is in scope, or that creates the scope root itself.
func createScope(s *ord.Scope, args *pyjson.Object) error {
	parent, _ := args.Get("parent_ord")
	name, _ := args.Get("name")
	_, err := s.CheckChild(parent, name)
	return err
}

// createPlanWith refuses to create a scope root that already exists before planning: the station
// would rename the new component on the collision and put it outside the scope.
func createPlanWith(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	parent, _ := args.Get("parent_ord")
	name, _ := args.Get("name")
	if root, _ := w.Scope.CheckChild(parent, name); root {
		_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
		if err != nil {
			return nil, err
		}
		if _, exists := box.Children(nodes, "").Get(str(args, "name")); exists {
			root, _ := ord.ChildOrd(str(args, "parent_ord"), str(args, "name")) // CheckChild already joined it
			return nil, plan.ToolError("the scope root " + pyjson.StrRepr(root) +
				" already exists: nothing to create (a second one would be renamed outside the write scope)")
		}
		p, err := createPlan(c, args)
		if err == nil {
			p.Notes = append(p.Notes, "This creates the write-scope root itself; the same scope cannot remove it again "+
				"(n4_remove_component scopes on parent_ord): remove it under a wider scope or by hand.")
		}
		return p, err
	}
	return createPlan(c, args)
}

// linkScope scopes a link on its target only: the link is stored on the target, the source is only
// read. The source must still be a plain slot ORD, since a chained ORD resolves past its text.
func linkScope(s *ord.Scope, args *pyjson.Object) error {
	src, _ := args.Get("source_ord")
	if !ord.PlainSlotOrd(src) {
		return &safety.Error{Msg: "source_ord: ord " + pyjson.Repr(src) + " " + safety.ReasonScopePlain +
			" (station:|slot:/<path>, nothing chained after the slot path)"}
	}
	tgt, _ := args.Get("target_ord")
	if err := s.Check(tgt); err != nil {
		return &safety.Error{Msg: "target_ord: " + err.Error() + " (a link is stored on its target; source_ord is only read)"}
	}
	return nil
}
