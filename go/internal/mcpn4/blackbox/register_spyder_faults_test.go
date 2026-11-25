package blackbox

// n4_register_bacnet_spyder when a read or a step fails: the field write is never sent on a guess,
// a refusal is told from an outcome the kit cannot know, and a half-configured device is never
// compared with the request (the verdict is unverified).

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// failedStep is observed.failed_step of a run that stopped.
func failedStep(out map[string]any) string { return str(Dig(out, "observed", "failed_step")) }

// lastResult is the journal's result line of the run.
func (s *spyderRig) lastResult(t *testing.T) map[string]any {
	t.Helper()
	lines := s.Lines("journal.jsonl")
	if len(lines) == 0 {
		t.Fatalf("no journal")
	}
	return lines[len(lines)-1]
}

// dropAction answers an invokeAction of action with an HTTP 500, as a dropped connection does: the
// station is never reached, so what it applied is unknown to the caller.
func (s *spyderRig) dropAction(action string) {
	s.Station.SetHook(func(frame map[string]any) (int, []byte) {
		for _, m := range objects(frame["m"]) {
			if arg, _ := Dig(m, "b", "scarg").(map[string]any); arg["a"] == action {
				return 500, nil
			}
		}
		return 0, nil
	})
}

func (s *spyderRig) objectIDSet(t *testing.T) bool {
	t.Helper()
	return nodeValue(s.Rig, s.device("$32016"), "config", "deviceObject", "objectId") == "device:2016"
}

func TestRegisterSpyderDoesNotSendTheFieldWriteWhenItCannotReadTheDevicesStatus(t *testing.T) {
	s := newSpyderRig(t, 5)
	armed, refused := false, false
	s.Station.SetFaults(boxstation.Faults{
		OnSync: func(_ *boxstation.Station, op map[string]any) {
			armed = armed || strings.HasSuffix(str(op["n"]), "objectId")
		},
		Refuse: func(_ *boxstation.Station, sck string, _ map[string]any) error {
			if sck != "loadSlots" || !armed || refused {
				return nil
			}
			refused = true // the one re-read before writeDeviceInstance; the read-back after it works
			return errors.New("status read timed out")
		}})
	out := s.Srv.RunWrite(registerTool, spyArgs)
	want := "could not read the device's status before writeDeviceInstance: "
	if failed := failedStep(out); !strings.Contains(failed, want) || !strings.Contains(failed, "status read timed out") ||
		!strings.HasSuffix(failed, "; the field write was not sent") {
		t.Errorf("failed_step = %q", failed)
	}
	if got := s.inv.get(); len(got) != 0 {
		t.Errorf("invoked %v after a status that could not be read", got)
	}
	if out["verdict"] != "unverified" || len(objects(out["inverse"])) != 1 || s.device("$32016") == "" {
		t.Errorf("verdict %v, inverse %v, device %q", out["verdict"], out["inverse"], s.device("$32016"))
	}
}

func TestRegisterSpyderRefusedStepStopsTheSequenceWithAnUnverifiedVerdict(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.Station.SetFaults(boxstation.Faults{Refuse: func(_ *boxstation.Station, sck string, arg map[string]any) error {
		for _, op := range objects(arg["ops"]) {
			if sck == "syncTo" && op["nm"] == "s" && strings.HasSuffix(str(op["n"]), "macAddress") {
				return errors.New("macAddress is read-only")
			}
		}
		return nil
	}})
	out := s.Srv.RunWrite(registerTool, spyArgs)
	failed := failedStep(out)
	for _, w := range []string{"step 3 of 6", "address/macAddress", "refused by the station", "macAddress is read-only", "no later step was sent"} {
		if !strings.Contains(failed, w) {
			t.Errorf("failed_step %q lacks %q", failed, w)
		}
	}
	if out["verdict"] != "unverified" || out["in_doubt"] != nil {
		t.Errorf("verdict %v, in_doubt %v", out["verdict"], out["in_doubt"])
	}
	if got := s.inv.get(); len(got) != 0 || s.objectIDSet(t) {
		t.Errorf("a later step ran: invoked %v, objectId set %v", got, s.objectIDSet(t))
	}
	if inverse := objects(out["inverse"]); len(inverse) != 1 || inverse[0]["n"] != "$32016" {
		t.Errorf("inverse = %v", out["inverse"])
	}
}

func TestRegisterSpyderTransportFailureOfTheFieldWriteIsInDoubtAndSaysTheControllerMayHoldTheInstance(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.dropAction("writeDeviceInstance")
	out := s.Srv.RunWrite(registerTool, spyArgs)
	failed := failedStep(out)
	for _, w := range []string{"step 5 of 6", "writeDeviceInstance", "may have been applied", "controller may already hold the new instance",
		"no later step was sent"} {
		if !strings.Contains(failed, w) {
			t.Errorf("failed_step %q lacks %q", failed, w)
		}
	}
	if strings.Contains(failed, "refused") {
		t.Errorf("a transport failure is not a refusal: %q", failed)
	}
	if out["in_doubt"] != true || out["verdict"] != "unverified" || s.lastResult(t)["in_doubt"] != true {
		t.Errorf("in_doubt %v, verdict %v, journal %v", out["in_doubt"], out["verdict"], s.lastResult(t))
	}
	if len(objects(out["inverse"])) != 1 || len(s.inv.get()) != 0 {
		t.Errorf("inverse %v, invoked %v", out["inverse"], s.inv.get())
	}
}

func TestRegisterSpyderTransportFailureOfAnotherStepIsInDoubtWithoutTheControllerSentence(t *testing.T) {
	s := newSpyderRig(t, 5)
	s.dropAction("submitPointDiscoveryJob")
	out := s.Srv.RunWrite(registerTool, spyArgs)
	failed := failedStep(out)
	if !strings.Contains(failed, "step 6 of 6") || !strings.Contains(failed, "may have been applied") ||
		strings.Contains(failed, "controller may already hold") || out["in_doubt"] != true {
		t.Errorf("failed_step %q, in_doubt %v", failed, out["in_doubt"])
	}
}
