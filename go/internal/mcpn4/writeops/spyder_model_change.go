package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// modelChange is the Spyder action the Workbench Controller Summary "Device Model" save invokes
// (BControllerSummaryView: modelChange(BVector{newModel: internal name})). The station sets
// controllerModel and compiledFwModel, drops Sylk devices the model does not support, reassigns
// the program's terminals and marks it modified (BDeviceUtil.doModelChange). Nothing reaches the
// controller until a download.
const (
	modelChange      = "modelChange"
	defaultSpyderMdl = "ModelBACnet3" // BBacnetSpyder.controllerModel default
)

type modelChangeData struct{ to, from spyderModel }

// modelChangePlan is the plan of modelChange to the display name in arg on a BacnetSpyder.
func modelChangePlan(ordText string, h any, nodes *pyjson.Object, args *pyjson.Object) (*plan.Planned, error) {
	if t := rootType(nodes); t != spyderType {
		return nil, plan.ToolError("action 'modelChange' needs a " + spyderType + ", got " + pyjson.StrRepr(t) + " at " + ordText)
	}
	raw, _ := args.Get("arg")
	name, _ := raw.(string)
	to, ok := spyderModelByName(name)
	if !ok {
		return nil, plan.ToolError("modelChange needs arg = a Spyder model display name (" + strings.Join(spyderModelNames(), ", ") +
			"), got " + pyjson.Repr(raw))
	}
	current := currentSpyderModel(nodes)
	from, ok := spyderModelByInternal(current)
	if !ok {
		return nil, plan.ToolError("cannot tell the current model of " + ordText + " (" + pyjson.Repr(current) + "); refusing modelChange")
	}
	if from.internal == to.internal {
		return nil, plan.ToolError(ordText + " is already configured as " + to.name)
	}
	targets := pyjson.NewObject()
	targets.Set(ordText, h)
	notes := []any{
		"changes the configured model from " + from.name + " (" + from.io + ") to " + to.name + " (" + to.io +
			"); the station reassigns the program's terminals, drops unsupported Sylk devices and marks the program modified",
		"station-side only: nothing reaches the controller until a full download; check the program's I/O fits the new model first",
		"inverse (partial): modelChange back to " + from.name + "; Sylk devices the station dropped and the terminal assignments it changed are not restored",
	}
	return &plan.Planned{Ops: []any{invokeOp(modelChangeArg(h, to))}, Inverse: []any{invokeOp(modelChangeArg(h, from))},
		Targets: targets, Notes: notes, Data: modelChangeData{to: to, from: from}}, nil
}

func modelChangeArg(h any, m spyderModel) *pyjson.Object {
	return obj("h", h, "a", modelChange, "b", obj("nm", "p", "t", "baja:Vector", "s",
		[]any{obj("nm", "p", "n", "newModel", "t", "baja:String", "v", m.internal)}))
}

// modelChangeReadback reads the configured model and compiledFwModel back.
func modelChangeReadback(c plan.Client, ordText string, p *plan.Planned, reply any) (any, any, any, string, error) {
	d := p.Data.(modelChangeData)
	requested := obj("action", modelChange, "model", d.to.name, "compiled_fw_model", d.to.latestID)
	_, nodes, err := plan.Handle(c, ordText, 2)
	if err != nil {
		return nil, nil, nil, "", err
	}
	fw, _ := wholeNumberText(box.ChildValue(box.Children(nodes, ""), "compiledFwModel", nil))
	current := currentSpyderModel(nodes)
	shown := current
	if m, ok := spyderModelByInternal(current); ok {
		shown = m.name
	}
	observed := obj("model", shown, "compiled_fw_model", fw)
	if fw == d.to.latestID && current == d.to.internal {
		return requested, reply, observed, "verified", nil
	}
	return requested, reply, observed, "mismatch", nil
}

// currentSpyderModel is the configured model: the loaded program's controllerModel, else the
// device's, else the BBacnetSpyder default (a default value is not sent).
func currentSpyderModel(nodes *pyjson.Object) string {
	for _, k := range []string{"ControlProgram/controllerModel", "controllerModel"} {
		if n := box.Node(nodes, k); n != nil {
			if v, ok := n.Get("v"); ok {
				return pyjson.Str(v)
			}
		}
	}
	return defaultSpyderMdl
}

func spyderModelByName(name string) (spyderModel, bool) {
	for _, m := range spyderModels {
		if strings.EqualFold(m.name, strings.TrimSpace(name)) {
			return m, true
		}
	}
	return spyderModel{}, false
}

func spyderModelByInternal(internal string) (spyderModel, bool) {
	for _, m := range spyderModels {
		if m.internal == internal {
			return m, true
		}
	}
	return spyderModel{}, false
}

func spyderModelNames() []string {
	var out []string
	for _, hw := range []int{31, 32, 33, 34, 51, 52, 53, 54, 55} {
		out = append(out, spyderModels[hw].name)
	}
	return out
}
