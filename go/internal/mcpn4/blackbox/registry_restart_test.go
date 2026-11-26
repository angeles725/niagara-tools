package blackbox

// A station restarted with a new or upgraded module (stop, install, start) while this server kept
// its session (nothing asked the station in between, so no lost-session reply came): the type
// registry this server read before still says the module's type is unknown. n4_create_component
// reads the registry again once before it refuses a type the cached registry does not describe
// (issue #1166), and the refusal says whether the registry it relied on was fresh.

import (
	"strings"
	"sync/atomic"
	"testing"
)

var (
	unknownPump = map[string]any{"vendorX:Pump": map[string]any{"c": "", "isErr": true, "m": "vendorX:Pump", "s": ""}}
	knownPump   = map[string]any{"vendorX:Pump": map[string]any{"c": []any{map[string]any{"st": "p", "n": "out"}}}}
)

// countRegistry counts the registry calls the station gets.
func countRegistry(r *Rig) *int32 { return failRegistry(r, 0) }

func TestACreateOfATypeAModuleDeployedAfterTheCachedRegistryIsPlannedWithoutReconnecting(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(unknownPump)
	args := withArgs(createPump, Args{"type": "vendorX:Pump"})
	refusedDry(t, r, "n4_create_component", args, "vendorX:Pump", "not known to the station")
	r.Station.SetRegistry(knownPump) // the station restarted with the module installed
	plan := r.Srv.Dry("n4_create_component", args)
	if plan["confirmation_token"] == nil {
		t.Errorf("no token for a type the station now describes: %v", plan)
	}
	if len(r.Children()) != 0 {
		t.Errorf("a dry run added %v", r.Children())
	}
}

func TestATypeStillUnknownAfterTheRegistryIsReadAgainKeepsTheRefusalAndSaysSo(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(unknownPump)
	calls := countRegistry(r)
	args := withArgs(createPump, Args{"type": "vendorX:Pump"})
	first := r.Srv.Err("n4_create_component", args)
	if !strings.Contains(first, "registry was read from the station for this call") || atomic.LoadInt32(calls) != 1 {
		t.Errorf("first refusal (registry calls %d): %s", atomic.LoadInt32(calls), first)
	}
	second := r.Srv.Err("n4_create_component", args)
	if !strings.Contains(second, "re-read from the station just now and still does not") || atomic.LoadInt32(calls) != 2 {
		t.Errorf("second refusal (registry calls %d): %s", atomic.LoadInt32(calls), second)
	}
	for _, text := range []string{first, second} {
		for _, want := range []string{"type vendorX:Pump is not known to the station: its type registry does not describe it",
			"registry may be stale", "touch the module jar", "reconnect with n4_connect"} {
			if !strings.Contains(text, want) {
				t.Errorf("refusal lacks %q: %s", want, text)
			}
		}
	}
	if lines := r.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

func TestACreateOfAKnownTypeReadsTheRegistryOnce(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(knownPump)
	calls := countRegistry(r)
	args := withArgs(createPump, Args{"type": "vendorX:Pump"})
	r.Srv.Dry("n4_create_component", args)
	r.Srv.Dry("n4_create_component", args)
	if n := atomic.LoadInt32(calls); n != 1 {
		t.Errorf("registry calls = %d, want 1: a known type is served from the session's registry", n)
	}
}
