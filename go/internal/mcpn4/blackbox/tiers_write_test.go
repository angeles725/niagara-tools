package blackbox

// The tier gate on write execution (port of test_tiers TestTierAUnchanged, TestTierB and
// TestTierC; the per-tool coverage is TestEveryWriteToolIsGatedByTheTier).

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

// reasonTier starts every tier refusal (tiers.ReasonTier, worded by the server's tier gate).
const reasonTier = "version tier refuses writes"

var createPump = Args{"parent_ord": Folder, "name": "Pump", "type": "kitControl:NumericConst"}

// tierRig is a writes-allowed rig on a station reporting version ("" = no version), with the
// extra opt-in flags, connected with a verified identity.
func tierRig(t *testing.T, version string, configure func(o *Options)) *Rig {
	r := NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		if configure != nil {
			configure(o)
		}
	})
	r.Station.SetVersion(version)
	r.ConnectVerified()
	return r
}

func outcomes(r *Rig) []any {
	return field(anySlice(r.Lines("audit.jsonl")), "outcome")
}

func anySlice(lines []map[string]any) []any {
	out := []any{}
	for _, l := range lines {
		out = append(out, l)
	}
	return out
}

func TestTierAWritesWithoutAnnotation(t *testing.T) {
	r := tierRig(t, "4.14.0.162", nil)
	if plan := r.Srv.Dry("n4_create_component", createPump); plan["tier_gate"] != nil || plan["confirmation_token"] == nil {
		t.Errorf("plan = %v", plan)
	}
	if out := r.Srv.RunWrite("n4_create_component", createPump); out["verdict"] != "verified" || !slices.Contains(r.Children(), "Pump") {
		t.Errorf("out = %v, children %v", out, r.Children())
	}
}

func TestDryRunIsAllowedAndSaysTheTierWouldBlock(t *testing.T) {
	r := tierRig(t, "4.15.3.28", nil)
	plan := r.Srv.Dry("n4_create_component", createPump)
	gate, _ := plan["tier_gate"].(map[string]any)
	reason, _ := gate["reason"].(string)
	if plan["confirmation_token"] == nil || gate["tier"] != "B" || gate["version"] != "4.15.3.28" ||
		gate["would_block"] != true || !strings.Contains(reason, "--allow-tier-b FakeStation") {
		t.Errorf("plan = %v", plan)
	}
	if inner, _ := plan["plan"].(map[string]any); inner == nil || inner["tier_gate"] != nil { // not what the token vouches for
		t.Errorf("plan.plan = %v", plan["plan"])
	}
}

func TestExecutionIsRefusedWithoutOptIn(t *testing.T) {
	r := tierRig(t, "4.15.3.28", nil)
	plan := r.Srv.Dry("n4_create_component", createPump)
	text := r.Srv.Err("n4_create_component", Execute(createPump, plan["confirmation_token"]))
	for _, needle := range []string{reasonTier, "--allow-tier-b FakeStation", "Workbench"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if slices.Contains(r.Children(), "Pump") || !reflect.DeepEqual(outcomes(r), []any{"planned", "refused"}) ||
		len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("children %v, audit %v, journal %v", r.Children(), outcomes(r), r.Lines("journal.jsonl"))
	}
}

func TestTierBOperatorOptInAllowsWrites(t *testing.T) {
	r := tierRig(t, "4.15.3.28", func(o *Options) { o.AllowTierB = []string{StationName} })
	if plan := r.Srv.Dry("n4_create_component", createPump); plan["tier_gate"] != nil || plan["confirmation_token"] == nil {
		t.Errorf("plan = %v", plan)
	}
	if out := r.Srv.RunWrite("n4_create_component", createPump); out["verdict"] != "verified" {
		t.Errorf("out = %v", out)
	}
}

func TestReadsAreUnaffected(t *testing.T) {
	r := tierRig(t, "4.15.3.28", nil)
	for _, tool := range []string{"n4_navigate", "n4_read_slots", "n4_list_links"} {
		r.Srv.OK(tool, Args{"ord": Folder})
	}
}

func TestExecutionIsRefusedWithTheProbeSteps(t *testing.T) {
	r := tierRig(t, "4.10.1.36", nil)
	plan := r.Srv.Dry("n4_create_component", createPump)
	if Dig(plan, "tier_gate", "tier") != "C" {
		t.Errorf("plan = %v", plan)
	}
	text := r.Srv.Err("n4_create_component", Execute(createPump, plan["confirmation_token"]))
	for _, needle := range []string{reasonTier, "--allow-tier-c FakeStation", "reg.loadContract", "scratch", "Workbench"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if slices.Contains(r.Children(), "Pump") {
		t.Errorf("children %v", r.Children())
	}
}

func TestTierBOptInDoesNotOpenTierC(t *testing.T) {
	r := tierRig(t, "4.10.1.36", func(o *Options) { o.AllowTierB = []string{StationName} })
	if got := Dig(r.Srv.Dry("n4_create_component", createPump), "tier_gate", "would_block"); got != true {
		t.Errorf("would_block %v", got)
	}
}

func TestUnknownVersionIsRefusedAsTierC(t *testing.T) {
	r := tierRig(t, "", nil)
	plan := r.Srv.Dry("n4_create_component", createPump)
	gate, _ := plan["tier_gate"].(map[string]any)
	if v, has := gate["version"]; gate["tier"] != "C" || !has || v != nil {
		t.Errorf("tier_gate = %v", gate)
	}
	text := r.Srv.Err("n4_create_component", Execute(createPump, plan["confirmation_token"]))
	if want := tierRefusal(StationName); text != want {
		t.Errorf("refusal\n got %s\nwant %s", text, want)
	}
}

func TestTierCOperatorOptInAllowsWrites(t *testing.T) {
	r := tierRig(t, "4.10.1.36", func(o *Options) { o.AllowTierC = []string{StationName} })
	if out := r.Srv.RunWrite("n4_create_component", createPump); out["verdict"] != "verified" {
		t.Errorf("out = %v", out)
	}
}

// tierRefusal is the tier-C refusal of a station with no version, as the server words it
// (test_retro.tier_refusal in the kit).
func tierRefusal(station string) string {
	return "version tier refuses writes: station " + station + " is tier C (version unknown (no oBIX " +
		"productVersion at /obix/about/)): writes need the probe sequence first (METHODOLOGY " +
		"section 5): `reg.loadContract` and `loadRoot` succeed and one harmless scratch write " +
		"(create, read back, remove in a scratch folder) is verified on this build. The server " +
		"has no probe tool. Routes: (1) dry runs stay allowed here, so the plan can be " +
		"reviewed now; (2) make the change in Workbench; (3) run that probe (e.g. in " +
		"Workbench or a PoC script), then restart the server with `--allow-tier-c " + station + "`."
}
