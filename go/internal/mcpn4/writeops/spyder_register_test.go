package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const bacnetNet = "station:|slot:/Drivers/BacnetNetwork"

// A display name a station can only store escaped (a leading digit, a space) is created under its
// SlotPath escape: "2012" is stored as "$32012", like the Spyders Workbench added; the plan says so.
func TestCreateEscapesADisplayNameTheStationStoresEscaped(t *testing.T) {
	for _, tc := range []struct{ name, stored string }{
		{"2012", "$32012"}, {"Room Temp", "Room$20Temp"}, {"Power", "Power"}, {"_x1", "_x1"},
	} {
		c := &loads{nodes: obj(bacnetNet, obj("", obj("h", "30")))}
		p, err := writeops.CreateComponent.Plan(c, obj("parent_ord", bacnetNet, "name", tc.name,
			"type", "honeywellBacnetSpyder:BacnetSpyder"))
		if err != nil {
			t.Fatalf("%q: %v", tc.name, err)
		}
		if got := js(t, p.Ops[0]); !strings.Contains(got, `"n": "`+tc.stored+`"`) {
			t.Errorf("%q: op = %s, want stored name %s", tc.name, got, tc.stored)
		}
		notes := js(t, p.Notes)
		if escaped := tc.name != tc.stored; escaped != strings.Contains(notes, "stored as") {
			t.Errorf("%q: notes = %s", tc.name, notes)
		}
	}
}

// Separators, the escape character itself, control and non-ASCII characters stay refused.
func TestCreateStillRefusesNamesThatCannotBeEscapedSafely(t *testing.T) {
	for _, bad := range []string{"", "a/b", "a|b", "$32012", "a\nb", "Cuboñ", ".."} {
		c := &loads{nodes: obj(bacnetNet, obj("", obj("h", "30")))}
		_, err := writeops.CreateComponent.Plan(c, obj("parent_ord", bacnetNet, "name", bad, "type", "baja:Folder"))
		if err == nil || !strings.Contains(err.Error(), "must be a plain slot name") {
			t.Errorf("%q: err = %v", bad, err)
		}
	}
}

// writeDeviceInstance is allowed only on a Honeywell BACnet Spyder: it writes the component's
// config/deviceObject/objectId into the controller found at the component's address. It has no
// inverse and stays unverified.
func TestWriteDeviceInstanceOnlyOnASpyder(t *testing.T) {
	spyder := bacnetNet + "/$32012"
	c := &loads{nodes: obj(spyder, obj("", obj("h", "77", "t", "honeywellBacnetSpyder:BacnetSpyder")))}
	p, err := writeops.InvokeAction.Plan(c, obj("ord", spyder, "action", "writeDeviceInstance"))
	if err != nil {
		t.Fatal(err)
	}
	if got := js(t, p.Ops[0]); got != `{"ssc": "invokeAction", "arg": {"h": "77", "a": "writeDeviceInstance"}}` {
		t.Errorf("op = %s", got)
	}
	if len(p.Inverse) != 0 || !strings.Contains(js(t, p.Notes), "objectId") {
		t.Errorf("inverse = %v notes = %s", p.Inverse, js(t, p.Notes))
	}
	other := bacnetNet + "/Dev"
	c = &loads{nodes: obj(other, obj("", obj("h", "78", "t", "bacnet:BacnetDevice")))}
	if _, err := writeops.InvokeAction.Plan(c, obj("ord", other, "action", "writeDeviceInstance")); err == nil ||
		!strings.Contains(err.Error(), "honeywellBacnetSpyder:BacnetSpyder") {
		t.Errorf("non-Spyder: err = %v", err)
	}
	if _, err := writeops.InvokeAction.Plan(c, obj("ord", other, "action", "writeDeviceInstance", "arg", 1)); err == nil {
		t.Errorf("an arg was accepted")
	}
}
