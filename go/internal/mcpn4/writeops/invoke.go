package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// AllowedActions is ALLOWED_ACTIONS: the actions n4_invoke_action may call in this version. ping
// (a driver network, device or point device ext checks its device; no setting changes) was added
// after the Python port; the differential declares that difference.
var AllowedActions = []string{"set", "active", "inactive", "auto", "ping", writeDeviceInstance, modelChange, spyderDownload, spyderCompile, spyderLoad}

// writeDeviceInstance is the Honeywell BACnet Spyder action that writes the component's
// config/deviceObject/objectId into the controller found at the component's address (MS/TP
// network and MAC), so it renumbers a controller that still has a duplicate factory instance
// (BBacnetSpyder.doWriteDeviceInstance; niagara-research B1165). Only spyderType may call it.
const (
	writeDeviceInstance = "writeDeviceInstance"
	spyderType          = "honeywellBacnetSpyder:BacnetSpyder"
	msgWriteInstance    = "no inverse: writeDeviceInstance writes config/deviceObject/objectId into the controller " +
		"at the component's address (field write). To undo, set objectId back and invoke it again. " +
		"The read-back asks the controller its instance (getOnlineDeviceInstanceNumber) and compares it with objectId"
)

// InvokeAction is n4_invoke_action: set (its previous fallback is the inverse) or a state
// change (no inverse, unverified).
var InvokeAction = plan.Impl{ScopeOrds: ordArg, Plan: invokePlan, PlanWith: invokePlanWith, Readback: invokeReadbackWithLoad}

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
	switch action {
	case modelChange:
		return modelChangePlan(ordText, h, nodes, args)
	case spyderDownload:
		return spyderDownloadPlan(ordText, h, nodes, args)
	case spyderCompile:
		return spyderCompilePlan(ordText, h, nodes, args)
	case spyderLoad:
		return spyderLoadPlan(ordText, h, nodes, args)
	}
	_, hasArg := args.Get("arg")
	_, hasArgType := args.Get("arg_type")
	if action != "set" {
		if hasArg || hasArgType {
			return nil, plan.ToolError("action " + pyjson.StrRepr(action) + " takes no arg")
		}
		var notes []any
		if takesOverride(action) && overrideTarget(c, rootType(nodes)) {
			arg.Set("b", defaultOverride())
			if overrideNeverExpires(nodes) {
				notes = append(notes, msgOverrideNoExpiry)
			}
		}
		note := "no inverse: " + action + " changes the control state and is not restored automatically"
		var data any
		switch action {
		case "ping":
			note = "no inverse: ping only asks the driver to check the device and changes no setting"
			data = readPing(c, ordText, nodes) // the health before; the read-back reads it again
		case writeDeviceInstance:
			if t := rootType(nodes); t != spyderType {
				return nil, plan.ToolError("action " + pyjson.StrRepr(action) + " needs a " + spyderType +
					", got " + pyjson.StrRepr(t) + " at " + ordText)
			}
			if err := instanceFaulted(ordText, nodes); err != nil {
				return nil, err
			}
			note = msgWriteInstance
		}
		return &plan.Planned{Ops: []any{invokeOp(arg)}, Inverse: []any{}, Targets: targets, Notes: append([]any{note}, notes...),
			Data: data}, nil
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

// rootType is the type of the loaded component itself ("" when the station did not send it).
func rootType(nodes *pyjson.Object) string {
	if root := box.Node(nodes, ""); root != nil {
		t, _ := get(root, "t").(string)
		return t
	}
	return ""
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
