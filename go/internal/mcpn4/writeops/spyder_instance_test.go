package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// asks is a station that answers loads and getOnlineDeviceInstanceNumber.
type asks struct {
	loads
	online any
	err    error
	sent   []string
}

func (a *asks) SSC(name string, arg any) (any, error) {
	action, _ := arg.(*pyjson.Object).Get("a")
	a.sent = append(a.sent, name+" "+pyjson.Str(action))
	return a.online, a.err
}

// After writeDeviceInstance the read-back asks the controller its instance
// (getOnlineDeviceInstanceNumber, live: {"t":"baja:Integer","v":"2012"}) and compares it with the
// configured objectId (retro 2026-11-17 R2).
func TestWriteDeviceInstanceReadsTheControllerInstanceBack(t *testing.T) {
	spyder := bacnetNet + "/$32012"
	for _, tc := range []struct {
		online           any
		err              error
		verdict, observe string
	}{
		{obj("t", "baja:Integer", "v", "2012"), nil, "verified", `"online_instance": 2012`},
		{obj("t", "baja:Integer", "v", "9999"), nil, "mismatch", `"online_instance": 9999`},
		{nil, &box.Error{Kind: box.KindBox, Msg: "timeout"}, "unverified", "timeout"},
	} {
		c := &asks{loads: loads{nodes: obj(spyder, obj("", obj("h", "77", "t", "honeywellBacnetSpyder:BacnetSpyder")),
			spyder+"/config/deviceObject", obj("", obj("h", "79"), "objectId", obj("t", "bacnet:BacnetObjectIdentifier", "v", "device:2012")))},
			online: tc.online, err: tc.err}
		args := obj("ord", spyder, "action", "writeDeviceInstance")
		p, err := writeops.InvokeAction.Plan(c, args)
		if err != nil {
			t.Fatal(err)
		}
		_, _, observed, verdict, err := writeops.InvokeAction.Readback(c, args, p, []any{nil}, p.Inverse)
		if err != nil || verdict != tc.verdict || !strings.Contains(js(t, observed), tc.observe) ||
			!strings.Contains(js(t, observed), `"configured_instance": 2012`) {
			t.Errorf("%v: verdict %q observed %s err %v", tc.online, verdict, js(t, observed), err)
		}
		if len(c.sent) != 1 || c.sent[0] != "invokeAction getOnlineDeviceInstanceNumber" {
			t.Errorf("sent %v", c.sent)
		}
	}
}
