package blackbox

// n4_register_bacnet_spyder: create the Spyder, set its address and instance, write the instance
// into the controller and discover its points, as one dry run and one confirmation token (retro
// R12 delta M9, issue #932).

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestRegisterSpyderPlansEveryStepUnderOneToken(t *testing.T) {
	s := newSpyderRig(t, 5)
	plan := s.Srv.Dry(registerTool, spyArgs)
	if tok, _ := plan["confirmation_token"].(string); tok == "" {
		t.Fatalf("no token: %v", plan)
	}
	ops := objects(Dig(plan, "plan", "ops"))
	if len(ops) != 1 || ops[0]["nm"] != "a" || ops[0]["n"] != "$32016" || Dig(ops[0], "b", "t") != spyderType {
		t.Errorf("ops = %v", ops)
	}
	var got []string
	for _, step := range stepsOf(t, plan) {
		text := func(k string) string { v, _ := step[k].(string); return v }
		got = append(got, strings.TrimSpace(text("op")+" "+text("slot")+text("action")+" "+text("value")))
	}
	want := []string{"set address/addressType 3", "set address/networkNumber 2", "set address/macAddress 0c",
		"set config/deviceObject/objectId device:2016", "invoke writeDeviceInstance", "invoke submitPointDiscoveryJob"}
	if !slices.Equal(got, want) {
		t.Errorf("follow_up = %q\nwant %q", got, want)
	}
	if inverse := objects(Dig(plan, "plan", "inverse")); len(inverse) != 1 || inverse[0]["nm"] != "v" || inverse[0]["n"] != "$32016" {
		t.Errorf("inverse = %v", Dig(plan, "plan", "inverse"))
	}
	if notes := pyjsonText(Dig(plan, "plan", "notes")); !strings.Contains(notes, "writeDeviceInstance has no inverse") ||
		!strings.Contains(notes, "stored as") {
		t.Errorf("notes = %s", notes)
	}
	if s.device("$32016") != "" || len(s.Lines("journal.jsonl")) != 0 || len(s.inv.get()) != 0 {
		t.Errorf("the dry run changed the station or the journal")
	}
}

func TestRegisterSpyderExecutesAndReadsEveryStepBack(t *testing.T) {
	s := newSpyderRig(t, 5)
	out := s.Srv.RunWrite(registerTool, spyArgs)
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	h := s.device("$32016")
	for path, want := range map[string]string{"address/addressType": "3", "address/networkNumber": "2",
		"address/macAddress": "0c", "config/deviceObject/objectId": "device:2016"} {
		if got := nodeValue(s.Rig, h, strings.Split(path, "/")...); got != want {
			t.Errorf("%s = %v, want %s", path, got, want)
		}
	}
	for path, want := range map[string]any{"configured_instance": 2016.0, "online_instance": 2016.0, "object_id": "device:2016",
		"discover.job_state": "success", "discover.found": 2.0, "discover.hardware.model": "PUB6438S"} {
		if got := Dig(out["observed"], strings.Split(path, ".")...); got != want {
			t.Errorf("observed %s = %v, want %v", path, got, want)
		}
	}
	if got := s.inv.get(); !reflect.DeepEqual(got, [][2]string{{"writeDeviceInstance", h}, {"submitPointDiscoveryJob", s.deviceChild(h, "points")},
		{"getOnlineDeviceInstanceNumber", h}}) {
		t.Errorf("invoked %v", got)
	}
	if got := phases(s.Rig); !reflect.DeepEqual(got, []any{"intent", "component-intent", "result"}) {
		t.Errorf("journal phases = %v", got)
	}
	if inverse := objects(out["inverse"]); len(inverse) != 1 || inverse[0]["n"] != "$32016" {
		t.Errorf("inverse = %v", out["inverse"])
	}
}

func TestRegisterSpyderWithoutDiscoveryAsksNoPoints(t *testing.T) {
	s := newSpyderRig(t, 5)
	out := s.Srv.RunWrite(registerTool, withArgs(spyArgs, Args{"discover": false}))
	if out["verdict"] != "verified" || Dig(out["observed"], "discover") != nil {
		t.Errorf("out = %v", out)
	}
	if got := s.inv.get(); len(got) != 2 || got[0][0] != "writeDeviceInstance" || got[1][0] != "getOnlineDeviceInstanceNumber" {
		t.Errorf("invoked %v", got)
	}
}

func TestRegisterSpyderModelCheckComparesTheDiscoveredHardware(t *testing.T) {
	s := newSpyderRig(t, 5)
	out := s.Srv.RunWrite(registerTool, withArgs(spyArgs, Args{"expected_model": "PUB6438S"}))
	if out["verdict"] != "verified" || Dig(out["observed"], "model_check") != "PUB6438S" {
		t.Errorf("matching model: %v", out)
	}
	s = newSpyderRig(t, 5)
	s.hardware = "54.0"
	out = s.Srv.RunWrite(registerTool, withArgs(spyArgs, Args{"expected_model": "PUB6438S"}))
	if hint := pyjsonText(Dig(out, "observed", "hint")); out["verdict"] != "mismatch" || !strings.Contains(hint, "PUB1012S") ||
		!strings.Contains(hint, "modelChange") {
		t.Errorf("other model: %v", out)
	}
}

// A new Spyder that starts in fault cannot take writeDeviceInstance (BOX: "raise() failed. Cannot find
// vmUuid"): the run refuses that step with the dry-run refusal's words and keeps the inverse.
func TestRegisterSpyderRefusesTheFieldWriteOnASpyderInFault(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.faulted = true
	out := s.Srv.RunWrite(registerTool, spyArgs)
	failed := pyjsonText(Dig(out, "observed", "failed_step"))
	if out["verdict"] != "unverified" || !strings.Contains(failed, "writeDeviceInstance refused") ||
		!strings.Contains(failed, "Device limit exceeded") {
		t.Errorf("out = %v", out)
	}
	if got := s.inv.get(); len(got) != 0 {
		t.Errorf("invoked %v after a refused step", got)
	}
	if len(objects(out["inverse"])) != 1 || s.device("$32016") == "" {
		t.Errorf("inverse %v, device %q", out["inverse"], s.device("$32016"))
	}
}

func TestRollbackOfARegistrationRemovesTheDevice(t *testing.T) {
	s := newSpyderRig(t, 5)
	out := s.Srv.RunWrite(registerTool, spyArgs)
	args := Args{"batch_id": out["batch_id"]}
	back := s.Srv.OK("n4_rollback", Execute(args, s.Srv.Dry("n4_rollback", args)["confirmation_token"]))
	if back["verdict"] != "verified" || s.device("$32016") != "" {
		t.Errorf("back = %v, device %q", back, s.device("$32016"))
	}
}
