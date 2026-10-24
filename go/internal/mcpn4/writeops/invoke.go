package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// AllowedActions is ALLOWED_ACTIONS: the actions n4_invoke_action may call in this version.
var AllowedActions = []string{"set", "active", "inactive", "auto"}

// InvokeAction is n4_invoke_action: set (its previous fallback is the inverse) or a state
// change (no inverse, unverified).
var InvokeAction = plan.Impl{ScopeOrds: ordArg, Plan: invokePlan, Readback: invokeReadback}

// fallbackTypes is _FALLBACK and argTypes _ARG_TYPES.
var (
	fallbackTypes = map[string]string{"NumericWritable": "baja:StatusNumeric", "BooleanWritable": "baja:StatusBoolean"}
	argTypes      = map[string]string{"baja:StatusNumeric": "baja:Double", "baja:StatusBoolean": "baja:Boolean"}
)

type invokeData struct {
	value  any
	fbType string
}

// fallbackType is _fallback_type: the type of the fallback slot, else the one the component's
// type implies ("" when neither tells).
func fallbackType(nodes *pyjson.Object) (string, error) {
	var t any
	if fb := box.Node(nodes, "fallback"); fb != nil {
		t, _ = fb.Get("t")
	} else {
		kind := any("")
		if root := box.Node(nodes, ""); root != nil {
			if v, ok := root.Get("t"); ok {
				kind = v
			}
		}
		s, ok := kind.(string)
		if !ok {
			return "", &plan.PyError{Type: "AttributeError", Msg: "'" + plan.TypeName(kind) + "' object has no attribute 'rpartition'"}
		}
		t = fallbackTypes[s[strings.LastIndex(s, ":")+1:]]
	}
	switch t.(type) {
	case []any, *pyjson.Object: // `in _ARG_TYPES` hashes it
		return "", &plan.PyError{Type: "TypeError", Msg: "cannot use '" + plan.TypeName(t) + "' as a dict key (unhashable type: '" + plan.TypeName(t) + "')"}
	}
	s, _ := t.(string)
	return s, nil
}

func invokePlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	action, ordText := str(args, "action"), str(args, "ord")
	if !allowed(action) {
		hint := ""
		if action == "save" {
			hint = "; use n4_save_station"
		}
		return nil, plan.ToolError("action " + pyjson.StrRepr(action) + " " + safety.ReasonAction + " (allowed: " +
			strings.Join(AllowedActions, ", ") + ")" + hint)
	}
	h, nodes, err := plan.Handle(c, ordText, 2)
	if err != nil {
		return nil, err
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	arg := pyjson.NewObject()
	arg.Set("h", h)
	arg.Set("a", action)
	_, hasArg := args.Get("arg")
	_, hasArgType := args.Get("arg_type")
	if action != "set" {
		if hasArg || hasArgType {
			return nil, plan.ToolError("action " + pyjson.StrRepr(action) + " takes no arg")
		}
		return &plan.Planned{Ops: []any{invokeOp(arg)}, Inverse: []any{}, Targets: targets,
			Notes: []any{"no inverse: " + action + " changes the control state and is not restored automatically"}}, nil
	}
	fbType, err := fallbackType(nodes)
	if err != nil {
		return nil, err
	}
	argType, ok := argTypes[fbType]
	if !ok {
		return nil, plan.ToolError("cannot tell the previous fallback of " + ordText + "; refusing set")
	}
	if given, _ := args.Get("arg_type"); given != argType || !hasArg {
		return nil, plan.ToolError("set on this component needs arg and arg_type " + argType)
	}
	raw, _ := args.Get("arg")
	value, err := plan.Coerce(argType, raw, "0")
	if err != nil {
		return nil, err
	}
	arg.Set("b", plan.ToBSON(argType, value))
	var previous any
	if box.Node(nodes, "fallback") != nil {
		if previous, err = plan.Observe(nodes, "fallback", fbType); err != nil {
			return nil, err
		}
	} else {
		var zero any = false
		if strings.HasSuffix(fbType, "Numeric") {
			zero = 0.0
		}
		p := pyjson.NewObject()
		p.Set("value", zero)
		p.Set("status", "0")
		previous = p
	}
	return &plan.Planned{Ops: []any{invokeOp(arg)}, Inverse: []any{setOp(h, "fallback", plan.ToBSON(fbType, previous))},
		Notes: []any{}, Targets: targets, Data: invokeData{value, fbType}}, nil
}

func allowed(action string) bool {
	for _, a := range AllowedActions {
		if a == action {
			return true
		}
	}
	return false
}

func invokeOp(arg *pyjson.Object) *pyjson.Object {
	o := pyjson.NewObject()
	o.Set("ssc", "invokeAction")
	o.Set("arg", arg)
	return o
}

func invokeReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies []any) (any, any, any, string, error) {
	action := str(args, "action")
	if action != "set" {
		requested := pyjson.NewObject()
		requested.Set("action", action)
		return requested, replies[0], nil, "unverified", nil
	}
	_, nodes, err := plan.Handle(c, str(args, "ord"), 2)
	if err != nil {
		return nil, nil, nil, "", err
	}
	d := p.Data.(invokeData)
	o, err := plan.Observe(nodes, "fallback", d.fbType)
	if err != nil {
		return nil, nil, nil, "", err
	}
	got, _ := o.(*pyjson.Object).Get("value")
	observed, requested := pyjson.NewObject(), pyjson.NewObject()
	observed.Set("fallback", got)
	requested.Set("fallback", d.value)
	return requested, replies[0], observed, verdict(observed, requested), nil
}
