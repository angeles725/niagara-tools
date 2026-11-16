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
