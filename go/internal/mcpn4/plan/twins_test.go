package plan_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/token"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// The Go twins of the test_tools_write tests marked go_twin("plan", ...) that this slice's
// tools reach; where the Python test drives n4_create_component (S11), the twin drives
// n4_set_slot through the same engine.

var batchIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestWriteOnAnUnverifiedSessionIsRefusedAndAudited(t *testing.T) {
	r := newRig(t)
	r.ctx.Session.IdentityVerified = false
	text, isErr, _ := r.call(t, "n4_set_slot", setArgs())
	if !isErr || !strings.Contains(text, "expected_station") || len(r.st.sent) != 0 {
		t.Fatalf("%v %q %d sent", isErr, text, len(r.st.sent))
	}
	audit := r.lines(t, "audit.jsonl")
	last := audit[len(audit)-1]
	if get(last, "outcome") != "refused" || get(last, "tool") != "n4_set_slot" {
		t.Fatal(pyjson.Repr(last))
	}
}

// TestAWriteWithoutASessionIsRefused, dry run or not, and audited (test_write_without_a_session_is_refused).
func TestAWriteWithoutASessionIsRefused(t *testing.T) {
	r := newRig(t)
	r.ctx.Session = nil
	for _, dry := range []bool{true, false} {
		args := setArgs()
		args.Set("dry_run", dry)
		if text, isErr, _ := r.call(t, "n4_set_slot", args); !isErr || text != "not connected: call n4_connect first" {
			t.Errorf("dry_run=%v: %v %q", dry, isErr, text)
		}
	}
	if audit := r.lines(t, "audit.jsonl"); len(audit) != 2 || get(audit[1], "outcome") != "refused" {
		t.Errorf("audit %d lines", len(audit))
	}
}

func TestDefaultConnectVerifiesIdentitySoWritesAreAllowed(t *testing.T) {
	srv := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{AllowWrites: true})
	srv.Connect(t)
	if !srv.Ctx.Session.IdentityVerified {
		t.Fatal("a default connect must verify the identity")
	}
}

func TestAnExpiredTokenIsRefused(t *testing.T) {
	r := newRig(t)
	now := 1000.0
	r.state.Tokens = token.New(300, func() float64 { return now })
	tok := r.dry(t, "n4_set_slot", setArgs())
	now = 4e9
	if text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), tok); !isErr || !strings.Contains(text, "expired") {
		t.Fatalf("%v %q", isErr, text)
	}
	if len(r.st.sent) != 0 {
		t.Fatal("an expired token sent an op")
	}
}

func TestThePlanNamesTheStationAndSessionItWasMadeOn(t *testing.T) {
	r := newRig(t)
	_, _, out := r.call(t, "n4_set_slot", setArgs())
	station := get(get(out, "plan").(*pyjson.Object), "station").(*pyjson.Object)
	want := `{'name': 'FakeStation', 'base_url': 'http://fake.invalid', 'session_id': '` + r.ctx.Session.SessionID + `'}`
	if pyjson.Repr(station) != want || !batchIDPattern.MatchString(r.ctx.Session.SessionID) {
		t.Fatalf("%s, want %s", pyjson.Repr(station), want)
	}
}

func TestANoneInverseIsNormalizedToAnEmptyList(t *testing.T) {
	r := newRig(t)
	impl := writeops.InvokeAction
	plan0 := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := plan0(c, args)
		if err == nil {
			p.Inverse = nil
		}
		return p, err
	}
	r.srv.Handlers["n4_invoke_action"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_invoke_action", impl)
	args := node("ord", sp, "action", "auto")
	_, _, out := r.execute(t, "n4_invoke_action", args, r.dry(t, "n4_invoke_action", args))
	result := r.lines(t, "journal.jsonl")[1]
	if pyjson.Repr(get(out, "inverse")) != "[]" || pyjson.Repr(get(result, "inverse")) != "[]" {
		t.Fatal(pyjson.Repr(out), pyjson.Repr(result))
	}
}

func TestAuditAndOutputsHoldNoSecretValue(t *testing.T) {
	r := newRig(t)
	r.ctx.SetSecret(readtoolstest.Password)
	args := setArgs()
	args.Set("password", "hunter2")
	args.Set("api_secret", "s3cr3t-x")
	_, _, plan := r.call(t, "n4_set_slot", args)
	tok := get(plan, "confirmation_token").(string)
	_, _, out := r.execute(t, "n4_set_slot", args, tok)
	r.execute(t, "n4_set_slot", args, tok)
	audit, _ := os.ReadFile(r.dir + "/audit.jsonl")
	for _, secret := range []string{"hunter2", "s3cr3t-x", tok, readtoolstest.Password} {
		if strings.Contains(string(audit), secret) {
			t.Fatalf("audit holds %q", secret)
		}
	}
	if strings.Contains(pyjson.Repr([]any{plan, out}), readtoolstest.Password) || !strings.Contains(string(audit), token.Redacted) {
		t.Fatal("secret in the outputs or nothing redacted")
	}
}

// TestRefusalsAndReadBackErrorsAreScrubbed: a refusal or read-back error that would echo the
// secret (a slot name, a station value) is scrubbed in the reply and the audit reason.
func TestRefusalsAndReadBackErrorsAreScrubbed(t *testing.T) {
	r := newRig(t)
	r.ctx.SetSecret("hunter2x")
	refused, _, _ := r.call(t, "n4_set_slot", node("ord", calc, "slot", "hunter2x/value", "value", 1, "value_type", "baja:Double"))
	r.st.onSend = func() { r.st.nodes[calc].Set("out/value", node("v", "hunter2x")) }
	_, _, out := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	if !strings.Contains(refused, "'***/value'") || get(out, "readback_error") != "ValueError: could not convert string to float: '***'" {
		t.Fatalf("%q %s", refused, pyjson.Repr(out))
	}
	for _, line := range r.lines(t, "audit.jsonl") { // args_redacted keeps the caller's own args, as in Python
		if reason := get(line, "reason").(string); strings.Contains(reason, "hunter2x") {
			t.Errorf("audit reason %q", reason)
		}
	}
}

// TestAWholeStationWriteNeedsAnyScope: a tool that names no ORD (save, rollback) runs only when
// at least one --write-scope is configured.
func TestAWholeStationWriteNeedsAnyScope(t *testing.T) {
	impl := writeops.InvokeAction
	impl.ScopeOrds = func(*pyjson.Object) []any { return nil }
	for _, scopes := range [][]string{{}, {"station:|slot:/Elsewhere"}} {
		r := newRig(t, scopes...)
		r.srv.Handlers["n4_invoke_action"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_invoke_action", impl)
		text, isErr, _ := r.call(t, "n4_invoke_action", node("ord", sp, "action", "auto"))
		if isErr != (len(scopes) == 0) || isErr && !strings.Contains(text, "no --write-scope configured") {
			t.Errorf("%v: %v %q", scopes, isErr, text)
		}
	}
}
