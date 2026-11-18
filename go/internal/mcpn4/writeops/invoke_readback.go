package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// invokeReadback is n4_invoke_action's read-back: the fallback for set, a typed check for the
// Spyder actions, and unverified for a plain state change.
func invokeReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	action := str(args, "action")
	if action != "set" {
		requested := pyjson.NewObject()
		requested.Set("action", action)
		switch action {
		case modelChange:
			return modelChangeReadback(c, str(args, "ord"), p, replies[0])
		case spyderDownload, spyderCompile:
			return spyderDownloadReadback(c, p, replies[0])
		}
		if action == writeDeviceInstance {
			ordText := str(args, "ord")
			h, _ := p.Targets.Get(ordText)
			return writeInstanceReadback(c, ordText, h, requested, replies[0])
		}
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
