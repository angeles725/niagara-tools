package session

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestTheLatchHoldsForTheCooldownAndCountsWholeSeconds(t *testing.T) {
	t.Parallel()
	ctx, c := newContext(Config{CredentialEnv: "LAB", AuthCooldown: 30})
	if ctx.AuthBlock() != "" || ctx.AuthRetryIn() != 0 {
		t.Fatal("a fresh context is latched")
	}
	ctx.SetSecret("pw-value")
	ctx.UseCredentials(ctx.CredentialsFingerprint("admin", "pw-value"))
	ctx.NoteAuthFailure(errors.New("HTTP 401 from station (pw-value)"))
	want := "station calls are paused after an authentication failure (HTTP 401 from station (***)): " +
		"retry in 30 s, or fix LAB_USER/LAB_PASSWORD and call n4_connect again; every rejected " +
		"call counts toward the station lock-out (5 failures in 30 s)"
	if got := ctx.AuthBlock(); got != want {
		t.Errorf("AuthBlock() = %q\nwant %q", got, want)
	}
	for _, step := range []struct {
		advance float64
		left    int
	}{{0.5, 30}, {28.6, 1}, {0.9, 0}} {
		c.now += step.advance
		if got := ctx.AuthRetryIn(); got != step.left {
			t.Errorf("after +%v: AuthRetryIn() = %d, want %d", step.advance, got, step.left)
		}
	}
	if ctx.AuthBlock() != "" || ctx.latch != nil {
		t.Error("the latch outlived its cooldown")
	}
}

func TestOnlyTheFailedCredentialsAreBlockedAtConnect(t *testing.T) {
	t.Parallel()
	ctx, _ := newContext(Config{CredentialEnv: "MCP_N4", AuthCooldown: 120})
	bad, good := ctx.CredentialsFingerprint("admin", "wrong"), ctx.CredentialsFingerprint("admin", "right")
	if bad == good || len(bad) != 64 || strings.Contains(bad, "wrong") {
		t.Fatalf("fingerprints %q %q", bad, good)
	}
	ctx.UseCredentials(bad)
	ctx.NoteAuthFailure(errors.New("HTTP 401"))
	if !strings.Contains(ctx.AuthBlockFor(bad), "retry in 120 s") {
		t.Errorf("the failed credentials are not blocked: %q", ctx.AuthBlockFor(bad))
	}
	if ctx.AuthBlockFor(good) != "" {
		t.Error("other credentials are blocked during the cooldown")
	}
	if ctx.AuthBlock() == "" {
		t.Error("a station call without credentials is not blocked")
	}
	ctx.ClearAuthLatch()
	if ctx.AuthBlock() != "" {
		t.Error("ClearAuthLatch left the latch")
	}
}

func TestAuthReportShowsThePauseOnlyWhileItHolds(t *testing.T) {
	t.Parallel()
	ctx, c := newContext(Config{AuthCooldown: 30})
	out := pyjson.NewObject()
	ctx.AddAuthReport(out)
	if out.Len() != 0 {
		t.Errorf("unlatched report %s", encode(t, out))
	}
	ctx.NoteAuthFailure(errors.New("HTTP 401 from station"))
	c.now += 0.25
	ctx.AddAuthReport(out)
	if got, want := encode(t, out), `{"auth_paused": {"reason": "HTTP 401 from station", "retry_in_s": 30}}`; got != want {
		t.Errorf("report %s, want %s", got, want)
	}
}
