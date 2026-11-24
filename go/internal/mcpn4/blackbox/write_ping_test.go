package blackbox

// n4_invoke_action ping: the read-back reads the health (lastOkTime, lastFailTime, lastFailCause)
// and status of the checked device, or of every device of a network, before and after the ping
// and reports what changed (retro R12 delta M8, #931).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	pingNet    = Folder + "/Net1"
	pingDevOrd = pingNet + "/Dev1"
	pingHeader = "Slot Path,Status Bits,Last Ok Time,Last Fail Time,Last Fail Cause\n"
	pingOld    = "18-Nov-26 11:44 PM CST"
	pingNew    = "24-Nov-26 9:00 AM CST"
)

// pingRig is a verified writes-allowed rig with a BACnet network (Net1) holding a device (Dev1)
// whose health and status the test sets, and servlets for the device BQL.
type pingRig struct {
	*Rig
	sv       *boxstation.Servlets
	net, dev string
}

func newPingRig(t *testing.T) *pingRig {
	t.Helper()
	p := &pingRig{}
	p.Rig = NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		p.sv = boxstation.ServeServlets(t, r.Station)
		o.Stations = map[string]string{StationName: p.sv.URL}
	})
	p.ConnectVerified()
	p.net = p.Add("3", "Net1", "bacnet:BacnetNetwork").Handle
	p.dev = p.Add(p.net, "Dev1", "bacnet:BacnetDevice").Handle
	return p
}

// slots gives the component with handle h plain slots (name, type, value).
func (p *pingRig) slots(h string, slots ...[3]string) {
	p.Station.Do(func(st *boxstation.Station) {
		n := st.ByHandle(h)
		for _, s := range slots {
			if c := n.Child(s[0]); c != nil {
				c.Value = s[2]
				continue
			}
			n.Children = append(n.Children, &boxstation.Node{Name: s[0], Type: s[1], Value: s[2]})
		}
	})
}

// health is the BPingHealth of the device with handle h.
func (p *pingRig) health(h, ok, fail, cause string) {
	p.Station.Do(func(st *boxstation.Station) { setHealth(st, h, ok, fail, cause) })
}

// setHealth is health for a caller that already holds the station (a hook).
func setHealth(st *boxstation.Station, h, ok, fail, cause string) {
	n := st.ByHandle(h)
	hl := n.Child("health")
	if hl == nil {
		hl = st.Component(n, "health", "driver:PingHealth", "")
	}
	set := func(name, typ, v string) {
		if c := hl.Child(name); c != nil {
			c.Value = v
		} else if v != "" {
			hl.Children = append(hl.Children, &boxstation.Node{Name: name, Type: typ, Value: v})
		}
	}
	set("lastOkTime", "baja:AbsTime", ok)
	set("lastFailTime", "baja:AbsTime", fail)
	set("lastFailCause", "baja:String", cause)
}

// afterPing runs fn once the station accepted a ping.
func (p *pingRig) afterPing(fn func(st *boxstation.Station, n *boxstation.Node)) {
	p.Station.Do(func(st *boxstation.Station) {
		st.AfterInvoke = func(st *boxstation.Station, action string, n *boxstation.Node) {
			if action == "ping" {
				fn(st, n)
			}
		}
	})
}

func (p *pingRig) deviceQuery(csv string) {
	routeOrds(p.sv, boxstation.OrdRoute{Needle: "driver:Device", Body: []byte(csv)})
}

func ping(p *pingRig, ord string) map[string]any {
	return p.Srv.RunWrite("n4_invoke_action", Args{"ord": ord, "action": "ping"})
}

func pingRows(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	return objects(Dig(out, "observed", "devices"))
}

func TestPingOfADeviceReportsTheHealthItMovedAndIsVerified(t *testing.T) {
	p := newPingRig(t)
	p.health(p.dev, pingOld, "", "")
	p.afterPing(func(st *boxstation.Station, _ *boxstation.Node) { setHealth(st, p.dev, pingNew, "", "") })
	out := ping(p, pingDevOrd)
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	want := map[string]any{"device": "slot:/Folder/Net1/Dev1",
		"before":  map[string]any{"status": []any{}, "last_ok": pingOld, "last_fail": nil, "last_fail_cause": nil},
		"after":   map[string]any{"status": []any{}, "last_ok": pingNew, "last_fail": nil, "last_fail_cause": nil},
		"changed": []any{"last_ok"}}
	if rows := pingRows(t, out); len(rows) != 1 || !reflect.DeepEqual(rows[0], want) {
		t.Errorf("devices = %v\nwant %v", rows, want)
	}
	for k, v := range map[string]any{"kind": "device", "devices_total": 1.0, "devices_changed": 1.0, "target": pingDevOrd} {
		if got := Dig(out, "observed", k); got != v {
			t.Errorf("observed.%s = %v, want %v", k, got, v)
		}
	}
	if note := Dig(out, "observed", "note"); note != nil {
		t.Errorf("note = %v on a ping that moved the health", note)
	}
}

func TestPingOfADeviceUnderADisabledNetworkNamesTheNetworkAndMovesNothing(t *testing.T) {
	p := newPingRig(t)
	p.slots(p.net, [3]string{"enabled", "baja:Boolean", "false"}, [3]string{"status", "baja:Status", "1"})
	p.slots(p.dev, [3]string{"status", "baja:Status", "1"})
	p.health(p.dev, pingOld, "", "")
	out := ping(p, pingDevOrd)
	if out["verdict"] != "mismatch" {
		t.Fatalf("out = %v", out)
	}
	rows := pingRows(t, out)
	if len(rows) != 1 || !reflect.DeepEqual(rows[0]["before"], rows[0]["after"]) || !reflect.DeepEqual(rows[0]["changed"], []any{}) ||
		!reflect.DeepEqual(Dig(rows[0], "after", "status"), []any{"disabled"}) {
		t.Errorf("devices = %v", rows)
	}
	if note, _ := Dig(out, "observed", "note").(string); !strings.Contains(note, "network is disabled") ||
		!strings.Contains(note, "no health slot moved") {
		t.Errorf("note = %q", note)
	}
	if Dig(out, "observed", "devices_changed") != 0.0 {
		t.Errorf("observed = %v", out["observed"])
	}
}

func TestPingOfADisabledOrDownDeviceNamesItsStatus(t *testing.T) {
	for flag, bits := range map[string]string{"disabled": "1", "down": "4"} {
		p := newPingRig(t)
		p.slots(p.dev, [3]string{"status", "baja:Status", bits})
		p.health(p.dev, pingOld, "", "")
		out := ping(p, pingDevOrd)
		note, _ := Dig(out, "observed", "note").(string)
		if out["verdict"] != "mismatch" || !strings.Contains(note, "device is "+flag) {
			t.Errorf("%s: verdict %v, note %q", flag, out["verdict"], note)
		}
	}
}

func TestPingOfAnEnabledOkDeviceThatMovedNothingSaysTheCheckMayStillRun(t *testing.T) {
	p := newPingRig(t)
	p.health(p.dev, pingOld, "", "")
	out := ping(p, pingDevOrd)
	note, _ := Dig(out, "observed", "note").(string)
	if out["verdict"] != "mismatch" || !strings.Contains(note, "may still be running") {
		t.Errorf("verdict %v, note %q", out["verdict"], note)
	}
}

func TestPingOfANetworkReadsEveryDeviceWithOneQueryAndListsTheChangedFirst(t *testing.T) {
	p := newPingRig(t)
	before := pingHeader +
		"slot:/Folder/Net1/Dev1,0," + pingOld + ",null,null\n" +
		"slot:/Folder/Net1/Dev2,0," + pingOld + ",null,null\n"
	after := pingHeader +
		"slot:/Folder/Net1/Dev1,0," + pingOld + ",null,null\n" +
		"slot:/Folder/Net1/Dev2,4," + pingOld + "," + pingNew + ",Timeout\n"
	p.deviceQuery(before)
	p.afterPing(func(*boxstation.Station, *boxstation.Node) { p.deviceQuery(after) })
	out := ping(p, pingNet)
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	rows := pingRows(t, out)
	if len(rows) != 2 || rows[0]["device"] != "slot:/Folder/Net1/Dev2" ||
		!reflect.DeepEqual(rows[0]["changed"], []any{"status", "last_fail", "last_fail_cause"}) ||
		!reflect.DeepEqual(Dig(rows[0], "after"), map[string]any{"status": []any{"down"}, "last_ok": pingOld,
			"last_fail": pingNew, "last_fail_cause": "Timeout"}) ||
		!reflect.DeepEqual(rows[1]["changed"], []any{}) {
		t.Errorf("devices = %v", rows)
	}
	if Dig(out, "observed", "kind") != "network" || Dig(out, "observed", "devices_total") != 2.0 ||
		Dig(out, "observed", "devices_changed") != 1.0 {
		t.Errorf("observed = %v", out["observed"])
	}
	queries := 0
	for _, o := range ordRequests(p.sv) {
		if strings.Contains(o, "driver:Device") {
			queries++
			if !strings.Contains(o, "Net1") {
				t.Errorf("query not based on the network: %s", o)
			}
		}
	}
	if queries != 3 { // the dry run, then the executing call's plan (before) and its read-back (after)
		t.Errorf("device queries = %d, want 3: one per read", queries)
	}
}

func TestPingOfADisabledNetworkNamesTheNetworkAsWhyNothingChanged(t *testing.T) {
	p := newPingRig(t)
	p.slots(p.net, [3]string{"enabled", "baja:Boolean", "false"}, [3]string{"status", "baja:Status", "1"})
	p.deviceQuery(pingHeader + "slot:/Folder/Net1/Dev1,1," + pingOld + ",null,null\n")
	out := ping(p, pingNet)
	note, _ := Dig(out, "observed", "note").(string)
	if out["verdict"] != "mismatch" || !strings.Contains(note, "network is disabled") ||
		Dig(out, "observed", "devices_changed") != 0.0 {
		t.Errorf("out = %v", out)
	}
}

func TestPingThatReadsNothingStaysUnverifiedAndSaysWhy(t *testing.T) {
	p := newPingRig(t) // no BQL route: the device query answers 404
	out := ping(p, pingNet)
	hint, _ := Dig(out, "observed", "hint").(string)
	if out["verdict"] != "unverified" || !strings.Contains(hint, "could not be read") {
		t.Errorf("out = %v", out)
	}
}
