package writeops_test

// Retro 2026-11-02 row 18: a point created in the same session and set at once. The set plans, runs
// and reads back verified like the same call on an existing point (the single and the writes form),
// provided the type registry describes the slot; a registry that does not names the cause.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/blackbox"
)

// TestMain lets the tests below start the real server over the fake station.
func TestMain(m *testing.M) { blackbox.Main(m) }

var createdPoint = map[string]any{"p": "baja:Component", "c": []any{
	map[string]any{"st": "p", "n": "fallback", "v": map[string]any{"nm": "p", "t": "baja:StatusNumeric", "d": "0.0 {ok}",
		"s": []any{map[string]any{"nm": "p", "n": "value", "t": "baja:Double", "v": "0.0"},
			map[string]any{"nm": "p", "n": "status", "t": "baja:Status", "v": "0"}}}},
	map[string]any{"st": "p", "n": "minActiveTime", "v": map[string]any{"nm": "p", "t": "baja:RelTime", "v": "0"}}}}

func createdRig(t *testing.T, registry map[string]any) *blackbox.Rig {
	t.Helper()
	r := blackbox.NewRig(t, func(r *blackbox.Rig, o *blackbox.Options) { r.WriteOptions(o) })
	r.ConnectVerified()
	r.Station.SetRegistry(registry)
	out := r.Srv.RunWrite("n4_create_component", blackbox.Args{"parent_ord": blackbox.Folder, "name": "Pt", "type": "kitBench:Pt"})
	if out["verdict"] != "verified" {
		t.Fatalf("create = %v", out)
	}
	return r
}

func TestASetOnAPointCreatedInTheSameSessionReadsBackVerified(t *testing.T) {
	r := createdRig(t, map[string]any{"kitBench:Pt": createdPoint, "baja:Component": map[string]any{}})
	ord := blackbox.Folder + "/Pt"
	for _, a := range []blackbox.Args{
		{"ord": ord, "slot": "fallback", "value": 3.0, "value_type": "baja:StatusNumeric"},
		{"ord": ord, "slot": "minActiveTime", "value": 5000, "value_type": "baja:RelTime"},
		{"writes": []any{
			map[string]any{"ord": ord, "slot": "fallback", "value": 4.0, "value_type": "baja:StatusNumeric"},
			map[string]any{"ord": ord, "slot": "minActiveTime", "value": 7000, "value_type": "baja:RelTime"}}},
	} {
		if plan := r.Srv.Dry("n4_set_slot", a); plan["confirmation_token"] == nil || blackbox.Dig(plan, "plan", "ops") == nil {
			t.Errorf("no plan for %v: %v", a, plan)
		}
		if out := r.Srv.RunWrite("n4_set_slot", a); out["verdict"] != "verified" {
			t.Errorf("%v: verdict %v in %v", a, out["verdict"], out)
		}
	}
}
