package blackbox

// A write executes only with a token a dry run issued in this server process and this connect
// session (retro 2026-11-16 row M17): the server-side half of the guard. Reconnect, single use
// and the transient-slot case are pinned in write_token_test.go and
// destructive_remove_volatile_test.go.

import (
	"strings"
	"testing"
)

func TestATokenFromAnEarlierServerProcessIsRefusedAfterARestart(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	r.Restart(nil) // same state dir and station: only the process, its key and its session are new
	r.ConnectVerified()
	text := r.Srv.Err("n4_create_component", Execute(createPump, token))
	if !strings.Contains(text, "does not match") || len(r.Children()) != 0 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

// A client cannot compute a token: its mac is keyed by a random per-process key, so a
// well-formed token no dry run issued is refused before any journal intent or station op.
func TestAWellFormedTokenNoDryRunIssuedIsRefused(t *testing.T) {
	r := verifiedRig(t)
	forged := "9999999999.0011223344556677." + strings.Repeat("ab", 32)
	text := r.Srv.Err("n4_create_component", Execute(createPump, forged))
	if !strings.Contains(text, "does not match") || len(r.Children()) != 0 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

// Retro mcpn4-history-batch-tools Δ3: after a reconnect the old token is still refused, and the
// refusal says how to get a new one (same arguments, dry_run=true) instead of only refusing. The
// re-plan is the operator's call: the server sends nothing and issues no token by itself.
func TestATokenFromAnEarlierSessionIsRefusedWithTheWayToReplan(t *testing.T) {
	r := verifiedRig(t)
	token := pumpToken(t, r)
	r.ConnectVerified() // same station, new session
	text := r.Srv.Err("n4_create_component", Execute(createPump, token))
	for _, part := range []string{"does not match", "station session", "n4_create_component", "same arguments", "dry_run=true"} {
		if !strings.Contains(text, part) {
			t.Errorf("missing %q: %s", part, text)
		}
	}
	if len(r.Children()) != 0 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("a refused execution sent ops: children %v", r.Children())
	}
	// The offered re-plan works: the same arguments with dry_run true give a token that executes.
	r.Srv.RunWrite("n4_create_component", createPump)
	if len(r.Children()) != 1 {
		t.Errorf("children after the re-planned execution = %v", r.Children())
	}
}

// A refusal for any other cause keeps its text: the re-plan note is only for a session change.
func TestAStaleStateRefusalDoesNotClaimASessionChange(t *testing.T) {
	r := verifiedRig(t)
	a := r.AddWithOut("A", 1)
	args := Args{"ord": Folder + "/A", "slot": "out", "value": 5.0, "value_type": "baja:StatusNumeric"}
	token := r.Srv.Dry("n4_set_slot", args)["confirmation_token"]
	r.Set(a.Handle, "out", StatusNumeric(3, "0"))
	if text := r.Srv.Err("n4_set_slot", Execute(args, token)); strings.Contains(text, "station session") {
		t.Errorf("refusal blames the session: %s", text)
	}
}
