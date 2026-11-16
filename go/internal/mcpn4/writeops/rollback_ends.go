package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// checkRestoredEnds checks every re-created component and both ends of every relink of every
// restored component (named as requested) under today's scope.
func checkRestoredEnds(w *plan.State, targets *pyjson.Object, ops, own, components, relinks []any) error {
	for _, s := range components {
		spec := s.(*pyjson.Object)
		top := ops[get(spec, "top").(int)].(*pyjson.Object)
		comp, err := childOrd(ordOf(targets, get(top, "h")), get(top, "n"))
		if err != nil {
			return err
		}
		end, err := ord.JoinOrd(comp, specPath(spec))
		if err != nil {
			return err
		}
		if err := w.Scope.Check(end); err != nil {
			return err
		}
	}
	for _, op := range own {
		comp, err := childOrd(ordOf(targets, get(op.(*pyjson.Object), "h")), get(op.(*pyjson.Object), "n"))
		if err != nil {
			return err
		}
		for _, spec := range relinks {
			if err := checkEnds(w.Scope, comp, spec.(*pyjson.Object)); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkEnds is _check_ends: both ends of a relink under scope.
func checkEnds(scope *ord.Scope, comp string, spec *pyjson.Object) error {
	for _, k := range []string{"source_path", "target_path"} {
		end, err := ord.JoinOrd(comp, get(spec, k).(string))
		if err != nil {
			return err
		}
		if err := scope.Check(end); err != nil {
			return err
		}
	}
	return nil
}

// childOrd is box.child_ord of a parent ORD and a recorded name of any type.
func childOrd(parent string, name any) (string, error) {
	s, ok := name.(string)
	if !ok {
		return "", &pyjson.ValueError{Msg: "ORD child name must be one plain slot name, got " + pyjson.Repr(name)}
	}
	return ord.ChildOrd(parent, s)
}

func specPath(spec *pyjson.Object) string {
	n, parent := get(spec, "n").(string), get(spec, "parent_path").(string)
	if parent == "" {
		return n
	}
	return parent + "/" + n
}
