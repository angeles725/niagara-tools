package blackbox

// A failed type-registry call is retried once (client A retro 2026-11-16 row M13): right after a
// module first loads, the first set of a slot still at its declared default failed once on the
// registry lookup and the retry worked. A request that gets no answer (an HTTP error, a transport
// failure) is asked again once; an answer, even one that says the type is unknown, is not.

import (
	"strings"
	"sync/atomic"
	"testing"
)

const registryNoAnswer = "the station's type registry did not answer (it may have just started): retry"

// failRegistry answers the first failures registry calls with HTTP 503 and counts every call.
func failRegistry(r *Rig, failures int32) *int32 {
	var calls int32
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		for _, m := range objects(frame["m"]) {
			if m["c"] == "reg" && atomic.AddInt32(&calls, 1) <= failures {
				return 503, []byte("starting")
			}
		}
		return 0, nil
	})
	return &calls
}

func TestASetOfADeclaredDefaultSurvivesOneFailedRegistryCall(t *testing.T) {
	r, c := dfxRig(t, verifiedRig)
	calls := failRegistry(r, 1)
	r.Srv.Dry("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "flag", "value": true, "value_type": "baja:Boolean"})
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("registry calls = %d, want the failed one and its retry", n)
	}
}

func TestTwoFailedRegistryCallsRefuseWithARetryHint(t *testing.T) {
	r, c := dfxRig(t, verifiedRig)
	calls := failRegistry(r, 2)
	text := r.Srv.Err("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "flag", "value": true, "value_type": "baja:Boolean"})
	if !strings.Contains(text, registryNoAnswer) {
		t.Errorf("reply %q lacks %q", text, registryNoAnswer)
	}
	if n := atomic.LoadInt32(calls); n != 2 {
		t.Errorf("registry calls = %d, want exactly one retry", n)
	}
	if lines := r.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

func TestAnUnknownTypeAnswerIsNotRetried(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(map[string]any{"vendorX:Pump": map[string]any{"c": "", "isErr": true, "m": "vendorX:Pump", "s": ""}})
	calls := failRegistry(r, 0)
	refusedDry(t, r, "n4_create_component", withArgs(createPump, Args{"type": "vendorX:Pump"}),
		"vendorX:Pump", "not known to the station", "registry may be stale")
	if n := atomic.LoadInt32(calls); n != 1 {
		t.Errorf("registry calls = %d, want 1: an unknown-type answer is an answer", n)
	}
}

func TestAStationErrorFromTheRegistryIsNotRetried(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	var calls int32
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		for _, m := range objects(frame["m"]) {
			if m["c"] == "reg" {
				atomic.AddInt32(&calls, 1)
				return 200, []byte(registryDown)
			}
		}
		return 0, nil
	})
	r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	if n := atomic.LoadInt32(&calls); n != 1 {
		t.Errorf("registry calls = %d, want 1: the station answered with its own error", n)
	}
}
