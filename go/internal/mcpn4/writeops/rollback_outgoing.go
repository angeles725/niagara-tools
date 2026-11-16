package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The plan side of the outgoing links a remove recorded (bench retro 2026-11-15 delta 20): links
// stored on components outside the removed subtree whose source was inside it.

// outgoingParts are the validated outgoing_link specs of a recorded inverse, in order.
func outgoingParts(inverse []any) ([]any, error) {
	var out []any
	for _, e := range inverse {
		if o, ok := e.(*pyjson.Object); ok {
			if v, has := o.Get("outgoing_link"); has {
				spec, err := outgoingSpec(v)
				if err != nil {
					return nil, err
				}
				out = append(out, spec)
			}
		}
	}
	return out, nil
}

// outgoingSpec is one journaled outgoing link: the target's plain slot ORD and handle, its slot,
// and the source's path inside the removed component and its slot.
func outgoingSpec(v any) (*pyjson.Object, error) {
	malformed := plan.ToolError("batch records a malformed outgoing link: refusing")
	spec, ok := v.(*pyjson.Object)
	if !ok || !ord.PlainSlotOrd(get(spec, "target")) {
		return nil, malformed
	}
	out := obj("target", get(spec, "target"))
	for _, k := range []string{"target_slot", "source_path", "source_slot"} {
		s, isStr := get(spec, k).(string)
		if !isStr || s == "" && k != "source_path" {
			return nil, malformed
		}
		if s != "" {
			for _, part := range strings.Split(s, "/") {
				if _, err := plan.Name(k, part); err != nil {
					return nil, err
				}
			}
		}
		out.Set(k, s)
	}
	if h, ok := get(spec, "target_h").(string); ok {
		out.Set("target_h", h)
	}
	return out, nil
}

// checkOutgoingEnds checks both ends of every outgoing link of every restored component (named as
// requested) under today's scope: the target outside it and the source inside the component.
func checkOutgoingEnds(scope *ord.Scope, targets *pyjson.Object, own, outgoing []any) error {
	for _, op := range own {
		comp, err := childOrd(ordOf(targets, get(op.(*pyjson.Object), "h")), get(op.(*pyjson.Object), "n"))
		if err != nil {
			return err
		}
		for _, s := range outgoing {
			if err := outgoingEnds(scope, comp, s.(*pyjson.Object)); err != nil {
				return err
			}
		}
	}
	return nil
}

// outgoingEnds checks the target and the source of one outgoing link of comp under scope.
func outgoingEnds(scope *ord.Scope, comp string, spec *pyjson.Object) error {
	if err := scope.Check(get(spec, "target")); err != nil {
		return err
	}
	src, err := ord.JoinOrd(comp, get(spec, "source_path").(string))
	if err != nil {
		return err
	}
	return scope.Check(src)
}
