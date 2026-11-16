package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/schedule"
)

// ReadSchedule is n4_read_schedule: one schedule component decoded into its plain model (one
// BOX load to schedule.LoadDepth). Slots the station leaves out because they hold their type's
// default are filled from the station's type registry (BOX reg/loadTypes), never guessed.
func (t *Tools) ReadSchedule(args *pyjson.Object) (*pyjson.Object, error) {
	ord, err := ordArg(args)
	if err != nil {
		return nil, err
	}
	nodes, elapsed, err := t.timedLoad(ord, schedule.LoadDepth)
	if err != nil {
		return nil, err
	}
	tl, ok := t.Ctx.Session.Client.(plan.TypeLoader)
	if !ok {
		return nil, toolError("this client cannot read the station's type registry, which " +
			"n4_read_schedule needs to fill the slots a station leaves out")
	}
	model, err := schedule.Read(nodes, func(typeSpec, name string) (*pyjson.Object, error) {
		return plan.DeclaredDefault(tl, typeSpec, name)
	})
	if err != nil {
		return nil, toolError("%s: %s", ord, err.Error())
	}
	out := obj("ord", ord)
	for _, k := range model.Keys() {
		v, _ := model.Get(k)
		out.Set(k, v)
	}
	out.Set("elapsed_ms", elapsed)
	return out, nil
}
