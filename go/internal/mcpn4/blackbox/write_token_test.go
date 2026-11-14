package blackbox

// The confirmation token and the write budget (port of test_tools_write TestTokenLayers,
// TestTokenStationBinding and TestBudget).

import (
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// pumpToken is the confirmation token of a dry run of createPump; it fails on an empty one.
func pumpToken(t *testing.T, r *Rig) string {
	t.Helper()
	token, _ := r.Srv.Dry("n4_create_component", createPump)["confirmation_token"].(string)
	if token == "" {
		t.Fatal("the dry run issued no token")
	}
	return token
}

func TestDryRunIsTheDefaultAndSendsNoMutation(t *testing.T) {
	r := verifiedRig(t)
	plan := r.Srv.OK("n4_create_component", createPump)
	wantOps := []any{map[string]any{"nm": "a", "h": "3", "n": "Pump",
		"b": map[string]any{"nm": "p", "t": "kitControl:NumericConst", "s": []any{}}}}
	inverse, _ := Dig(plan, "plan", "inverse").([]any)
	hash, _ := plan["plan_hash"].(string)
	token, _ := plan["confirmation_token"].(string)
	expires, _ := plan["expires_at"].(float64)
	if plan["dry_run"] != true || len(r.Children()) != 0 || Dig(plan, "plan", "tool") != "n4_create_component" ||
		!reflect.DeepEqual(Dig(plan, "plan", "ops"), wantOps) || len(inverse) == 0 || Dig(inverse[0], "nm") != "v" ||
		!regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(hash) || token == "" || expires <= 0 {
		t.Errorf("plan = %v, children %v", plan, r.Children())
	}
	if lines := r.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

func TestExecuteWithoutATokenIsRefused(t *testing.T) {
	r := verifiedRig(t)
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"dry_run": false}))
	if !strings.Contains(text, "confirmation_token") || len(r.Children()) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestATokenIsSingleUse(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	r.Srv.OK("n4_create_component", Execute(createPump, token))
	text := r.Srv.Err("n4_create_component", Execute(createPump, token))
	if !strings.Contains(text, "already used") || !reflect.DeepEqual(r.Children(), []string{"Pump"}) {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestATamperedTokenIsRefused(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	last := "0"
	if strings.HasSuffix(token, "0") {
		last = "1"
	}
	r.Srv.Err("n4_create_component", Execute(createPump, token[:len(token)-1]+last))
	if len(r.Children()) != 0 {
		t.Errorf("children %v", r.Children())
	}
}

func TestArgsChangedAfterTheTokenWasIssuedAreRefused(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	text := r.Srv.Err("n4_create_component", Execute(withArgs(createPump, Args{"name": "Other"}), token))
	if !strings.Contains(text, "does not match") || len(r.Children()) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestATokenForAnotherToolIsRefused(t *testing.T) {
	r := verifiedRig(t)
	r.AddWithOut("A", 1)
	token := pumpToken(t, r)
	r.Srv.Err("n4_set_slot", Execute(Args{"ord": Folder + "/A", "slot": "out", "value": 2.0,
		"value_type": "baja:StatusNumeric"}, token))
}

func TestStationStateChangedSinceTheDryRunIsRefused(t *testing.T) {
	r := verifiedRig(t)
	a := r.AddWithOut("A", 1)
	args := Args{"ord": Folder + "/A", "slot": "out", "value": 5.0, "value_type": "baja:StatusNumeric"}
	token := r.Srv.Dry("n4_set_slot", args)["confirmation_token"]
	r.Set(a.Handle, "out", StatusNumeric(3, "0")) // someone else writes
	if text := r.Srv.Err("n4_set_slot", Execute(args, token)); !strings.Contains(text, "this plan") {
		t.Errorf("refusal = %s", text)
	}
}

func TestATokenFromAnEarlierSessionIsRefusedAfterAReconnect(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	r.ConnectVerified() // same station, new session
	text := r.Srv.Err("n4_create_component", Execute(createPump, token))
	if !strings.Contains(text, "does not match") || len(r.Children()) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestATokenFromStationACannotExecuteOnStationB(t *testing.T) {
	other := boxstation.New(User, Password, "Other").Start(t)
	r := writeRig(t, func(o *Options) { o.Stations["Other"] = other.URL })
	r.ConnectVerified()
	token := pumpToken(t, r)
	r.Srv.OK("n4_connect", Args{"station": "Other"})
	text := r.Srv.Err("n4_create_component", Execute(createPump, token))
	var otherChildren int
	other.Do(func(st *boxstation.Station) { otherChildren = len(st.ByHandle("3").Children) })
	if !strings.Contains(text, "does not match") || otherChildren != 0 {
		t.Errorf("refusal = %s, Other's Folder has %d children", text, otherChildren)
	}
}

func TestWritesBeyondTheSessionBudgetAreRefused(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"}))
	if !strings.Contains(text, "budget") || !reflect.DeepEqual(r.Children(), []string{"A"}) {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

// F5 (audit 2026-10-03): the budget is per server process. A per-session budget let a reconnect
// reset --max-writes, so it bounded nothing across sessions.
func TestReconnectingKeepsTheProcessBudget(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	r.ConnectVerified()
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"}))
	if !strings.Contains(text, "write budget exhausted") || !strings.Contains(text, "server process") ||
		!reflect.DeepEqual(r.Children(), []string{"A"}) {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}
