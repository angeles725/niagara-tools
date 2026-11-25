package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const gw = "station:|slot:/Drivers/ModbusAsyncNetwork/Gw"

// ping asks the driver to check a device: no argument, no inverse, and a note saying it changes no setting.
func TestPingIsAllowedWithoutArgumentAndChangesNoSetting(t *testing.T) {
	c := &loads{nodes: obj(gw, obj("", obj("h", "3", "t", "modbusAsync:ModbusAsyncDevice")))}
	p, err := writeops.InvokeAction.Plan(c, obj("ord", gw, "action", "ping"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Ops); got != `[{"ssc": "invokeAction", "arg": {"h": "3", "a": "ping"}}]` {
		t.Errorf("ops = %s", got)
	}
	if got := js(t, p.Inverse); got != `[]` {
		t.Errorf("inverse = %s", got)
	}
	if got := js(t, p.Notes); got != `["no inverse: ping only asks the driver to check the device and changes no setting"]` {
		t.Errorf("notes = %s", got)
	}
	if _, err := writeops.InvokeAction.Plan(c, obj("ord", gw, "action", "ping", "arg", "x")); err == nil || err.Error() != "action 'ping' takes no arg" {
		t.Errorf("ping with arg: err = %v", err)
	}
}

// The other actions keep their note and the refusal lists ping.
func TestTheStateActionsKeepTheirNoteAndTheRefusalListsPing(t *testing.T) {
	c := &loads{nodes: obj(gw, obj("", obj("h", "3", "t", "control:BooleanWritable")))}
	p, err := writeops.InvokeAction.Plan(c, obj("ord", gw, "action", "auto"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Notes); got != `["no inverse: auto changes the control state and is not restored automatically"]` {
		t.Errorf("notes = %s", got)
	}
	_, err = writeops.InvokeAction.Plan(c, obj("ord", gw, "action", "discover"))
	if want := "action 'discover' is not allowed in this version (allowed: set, active, inactive, auto, ping, writeDeviceInstance, modelChange, spyderDownload, compile, loadControlProgram)"; err == nil || err.Error() != want {
		t.Errorf("err = %v, want %q", err, want)
	}
}
