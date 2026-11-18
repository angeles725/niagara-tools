package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

func spyderNodes(fw string) *loads {
	spyder := bacnetNet + "/$32012"
	return &loads{nodes: obj(spyder, obj("", obj("h", "77", "t", "honeywellBacnetSpyder:BacnetSpyder"),
		"compiledFwModel", obj("t", "baja:Integer", "v", fw),
		"ControlProgram", obj("h", "78", "t", "honeywellSpyderTool:ApplicationLogic"),
		"ControlProgram/controllerModel", obj("t", "baja:String", "v", map[string]string{"13": "ModelBACnet3", "8": "ModelMicroBACnet4"}[fw])))}
}

// modelChange (the Workbench Controller Summary "Device Model" save) takes the model's display
// name, sends BBacnetSpyder.modelChange a baja:Vector {newModel: the internal model name}, and its
// inverse changes back to the current model. It is a station-side change: nothing reaches the
// controller until a download.
func TestModelChangePlansTheStationActionWithItsInverse(t *testing.T) {
	spyder := bacnetNet + "/$32012"
	p, err := writeops.InvokeAction.Plan(spyderNodes("13"), obj("ord", spyder, "action", "modelChange", "arg", "PUB1012S"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"ssc": "invokeAction", "arg": {"h": "77", "a": "modelChange", "b": {"nm": "p", "t": "baja:Vector", "s": [{"nm": "p", "n": "newModel", "t": "baja:String", "v": "ModelMicroBACnet4"}]}}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op = %s", got)
	}
	if got := js(t, p.Inverse); !strings.Contains(got, `"v": "ModelBACnet3"`) {
		t.Errorf("inverse = %s", got)
	}
	notes := js(t, p.Notes)
	for _, part := range []string{"PUB6438S", "PUB1012S", "1 Universal Inputs", "download"} {
		if !strings.Contains(notes, part) {
			t.Errorf("notes miss %q: %s", part, notes)
		}
	}
	for _, bad := range []any{nil, "PUB9999", 54} {
		args := obj("ord", spyder, "action", "modelChange")
		if bad != nil {
			args.Set("arg", bad)
		}
		if _, err := writeops.InvokeAction.Plan(spyderNodes("13"), args); err == nil {
			t.Errorf("arg %v accepted", bad)
		}
	}
	if _, err := writeops.InvokeAction.Plan(spyderNodes("13"), obj("ord", spyder, "action", "modelChange", "arg", "PUB6438S")); err == nil ||
		!strings.Contains(err.Error(), "already") {
		t.Errorf("same model: err = %v", err)
	}
	other := &loads{nodes: obj(bacnetNet+"/Dev", obj("", obj("h", "9", "t", "bacnet:BacnetDevice")))}
	if _, err := writeops.InvokeAction.Plan(other, obj("ord", bacnetNet+"/Dev", "action", "modelChange", "arg", "PUB1012S")); err == nil {
		t.Errorf("non-Spyder accepted")
	}
}

// The read-back reads the model and compiledFwModel the station set (the new model's latest
// model id: 8 for PUB1012S, 13 for PUB6438S).
func TestModelChangeReadsTheModelBack(t *testing.T) {
	spyder := bacnetNet + "/$32012"
	args := obj("ord", spyder, "action", "modelChange", "arg", "PUB1012S")
	p, err := writeops.InvokeAction.Plan(spyderNodes("13"), args)
	if err != nil {
		t.Fatal(err)
	}
	for fw, verdict := range map[string]string{"8": "verified", "13": "mismatch"} {
		_, _, o, v, err := writeops.InvokeAction.Readback(spyderNodes(fw), args, p, []any{nil}, p.Inverse)
		if err != nil || v != verdict {
			t.Errorf("fw %s: verdict %q observed %s err %v", fw, v, js(t, o), err)
		}
	}
}
