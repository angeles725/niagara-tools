package writeops

import (
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// onlineInstanceAction is the Spyder action that asks the controller at the component's address
// for its device instance (live 2026-11-18: {"t":"baja:Integer","v":"2012"}); it changes nothing.
const onlineInstanceAction = "getOnlineDeviceInstanceNumber"

// writeInstanceReadback checks writeDeviceInstance: the instance the controller now reports must
// equal the component's config/deviceObject/objectId (retro 2026-11-17 R2). A failed question
// leaves the verdict unverified with why.
func writeInstanceReadback(c plan.Client, ordText string, h any, requested *pyjson.Object, reply any) (any, any, any, string, error) {
	observed := pyjson.NewObject()
	nodes, err := c.LoadTree(ordText+"/config/deviceObject", 1, box.LoadOptions{})
	if err != nil {
		return nil, nil, nil, "", err
	}
	configured := instance(box.ChildValue(box.Children(nodes, ""), "objectId", nil))
	observed.Set("configured_instance", configured)
	arg := pyjson.NewObject()
	arg.Set("h", h)
	arg.Set("a", onlineInstanceAction)
	answer, err := c.SSC("invokeAction", arg)
	if err != nil {
		observed.Set("hint", onlineInstanceAction+" failed: "+err.Error()+"; check with ping or a fresh Discover")
		return requested, reply, observed, "unverified", nil
	}
	v := answer
	if o, ok := answer.(*pyjson.Object); ok {
		v = get(o, "v")
	}
	online, err := strconv.Atoi(pyjson.Str(v))
	if err != nil {
		observed.Set("hint", onlineInstanceAction+" answered "+pyjson.Str(v)+", not an instance number")
		return requested, reply, observed, "unverified", nil
	}
	observed.Set("online_instance", online)
	if configured == nil || online != configured {
		return requested, reply, observed, "mismatch", nil
	}
	return requested, reply, observed, "verified", nil
}
