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

// faulted is a Spyder at 2012 whose own status and faultCause slots are as given ("" leaves the
// slot out, as a station omits a default).
func faulted(status, cause string) *asks {
	spyder := bacnetNet + "/$32012"
	root := obj("", obj("h", "77", "t", "honeywellBacnetSpyder:BacnetSpyder"))
	if status != "" {
		root.Set("status", obj("t", "baja:Status", "v", status))
	}
	if cause != "" {
		root.Set("faultCause", obj("t", "baja:String", "v", cause))
	}
	return &asks{loads: loads{nodes: obj(spyder, root)}}
}

// A faulted Spyder is refused at the dry run with its faultCause: the write would end in the BOX
// error "raise() failed. Cannot find vmUuid" and leave the controller as it was (retro R12 M4,
// batch 11768118 left in doubt by a licence capacity fault).
func TestWriteDeviceInstanceIsRefusedOnAFaultedSpyderWithItsCause(t *testing.T) {
	spyder := bacnetNet + "/$32012"
	args := obj("ord", spyder, "action", "writeDeviceInstance")
	for _, tc := range []struct{ status, cause, want string }{
		{"2", "Device limit exceeded", "writeDeviceInstance refused: " + spyder + " is in fault (faultCause: Device limit " +
			"exceeded). The write would fail with \"raise() failed. Cannot find vmUuid\" and the controller would not be " +
			"written. Clear the fault, then redo the dry run"},
		{"2;activeLevel=e", "", "writeDeviceInstance refused: " + spyder + " is in fault (faultCause is empty). The write " +
			"would fail with \"raise() failed. Cannot find vmUuid\" and the controller would not be written. Clear the " +
			"fault, then redo the dry run"},
	} {
		p, err := writeops.InvokeAction.Plan(faulted(tc.status, tc.cause), args)
		if err == nil || err.Error() != tc.want || p != nil {
			t.Errorf("status %q: plan %v, err %v\nwant %s", tc.status, p, err, tc.want)
		}
	}
}

// Any other status plans as before: a down or disabled Spyder, one with no status slot (ok).
func TestWriteDeviceInstanceStillPlansOnASpyderThatIsNotInFault(t *testing.T) {
	args := obj("ord", bacnetNet+"/$32012", "action", "writeDeviceInstance")
	for _, status := range []string{"", "0", "1", "4"} {
		p, err := writeops.InvokeAction.Plan(faulted(status, "left over from an old fault"), args)
		if err != nil || len(p.Ops) != 1 || !strings.Contains(js(t, p.Notes), "no inverse: writeDeviceInstance") {
			t.Errorf("status %q: %v %v", status, p, err)
		}
	}
}
