package token

// Go twins of test_safety.TestConfirmationTokens (go_server.go_twin("token", "injection")) and
// of the two redaction tests of test_safety.TestAuditLog: the clock, key and nonce are injected
// in process, which only a Go test can do to the Go server.

import (
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

type fixture struct {
	t    *testing.T
	now  float64
	tok  *Tokens
	args *pyjson.Object
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{t: t, now: 1000}
	f.tok = New(300, func() float64 { return f.now })
	f.args = objOf("ord", "station:|slot:/A", "slot", "out")
	return f
}

func objOf(kv ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

func (f *fixture) issue() string {
	f.t.Helper()
	tok, _, err := f.tok.Issue("n4_set_slot", f.args, "p1")
	if err != nil {
		f.t.Fatal(err)
	}
	return tok
}

func (f *fixture) consume(token any) error { return f.tok.Consume("n4_set_slot", f.args, "p1", token) }

// refused requires a *safety.Error whose text is exactly want.
func refused(t *testing.T, err error, want string) {
	t.Helper()
	se, ok := err.(*safety.Error)
	if !ok || se.Msg != want {
		t.Fatalf("got %#v, want the refusal %q", err, want)
	}
}

const (
	malformed = safety.ReasonTokenMissing + " or malformed: run a dry run first and pass the token it returns"
	mismatch  = safety.ReasonTokenMismatch + ", these arguments and this plan: run the dry run again"
	expired   = safety.ReasonTokenExpired + ": run the dry run again"
	reused    = safety.ReasonTokenReused + ": run the dry run again"
)

func TestIssueReturnsTokenAndExpiryFromTtl(t *testing.T) {
	f := newFixture(t)
	f.now = 1000.9 // int(now) truncates
	tok, expiresAt, err := f.tok.Issue("n4_set_slot", f.args, "p1")
	if err != nil || expiresAt != 1300 || !strings.HasPrefix(tok, "1300.") {
		t.Fatalf("got %q %d %v", tok, expiresAt, err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 || len(parts[1]) != 16 || len(parts[2]) != 64 {
		t.Fatalf("token shape %q: want <expiry>.<16 hex nonce>.<64 hex mac>", tok)
	}
}

func TestValidTokenIsAcceptedOnce(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	if err := f.consume(tok); err != nil {
		t.Fatal(err)
	}
	refused(t, f.consume(tok), reused)
}

func TestMissingOrMalformedTokenIsRefused(t *testing.T) {
	f := newFixture(t)
	for _, bad := range []any{nil, "", "garbage", "1.2", 5, pyjson.IntLiteral("5"), "1.2.3.4", ".a.b"} {
		refused(t, f.consume(bad), malformed)
	}
}

func TestNonAsciiOrOversizedTokenPartsRaiseOnlySafetyError(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	p := strings.Split(tok, ".")
	for _, bad := range []string{"²." + p[1] + "." + p[2], p[0] + "." + p[1] + ".éabc",
		p[0] + ".é." + p[2], "٣١.a.b", strings.Repeat("9", 400) + ".a.b", "-5.a.b",
		tok + "Ⅶ", "1234567890123456.a.b", "+5.a.b", " 5.a.b", p[0] + "." + p[1] + "\xed\xa0\x80." + p[2]} {
		refused(t, f.consume(bad), malformed)
	}
	if err := f.consume(tok); err != nil { // the genuine token is still spendable
		t.Fatal(err)
	}
}

func TestExpiredTokenIsRefused(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	f.now = 1300
	refused(t, f.consume(tok), expired)
	f.now = 1000 // an expired token is dropped: it never becomes spendable again
	refused(t, f.consume(tok), reused)
}

func TestTokenStillValidJustBeforeExpiry(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	f.now = 1299.999
	if err := f.consume(tok); err != nil {
		t.Fatal(err)
	}
}

func TestTamperedTokenIsRefused(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	last := "0"
	if strings.HasSuffix(tok, "0") {
		last = "1"
	}
	refused(t, f.consume(tok[:len(tok)-1]+last), mismatch)
	refused(t, f.consume(strings.ToUpper(tok)), mismatch) // the mac is compared as text
	if err := f.consume(tok); err != nil {
		t.Fatal(err)
	}
}

func TestTamperedExpiryIsRefused(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	expiry, rest, _ := strings.Cut(tok, ".")
	n, _ := strconv.Atoi(expiry)
	refused(t, f.consume(strconv.Itoa(n+9999)+"."+rest), mismatch)
	// Leading zeros parse to the same expiry, so the mac still matches (Python's int()).
	if err := f.consume("000" + tok); err != nil {
		t.Fatal(err)
	}
}

func TestTokenIsBoundToToolArgsAndPlanHash(t *testing.T) {
	f := newFixture(t)
	for _, c := range []struct {
		tool, plan string
		args       *pyjson.Object
	}{{"n4_create_link", "p1", f.args}, {"n4_set_slot", "p1", objOf("ord", "station:|slot:/B")},
		{"n4_set_slot", "p2", f.args}, {"n4_set_slot", "p1", objOf("ord", "station:|slot:/A", "slot", "out", "x", nil)}} {
		tok := f.issue()
		refused(t, f.tok.Consume(c.tool, c.args, c.plan, tok), mismatch)
	}
}

func TestDryRunAndTokenArgsAreExcludedFromTheBinding(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	args := objOf("dry_run", false, "ord", "station:|slot:/A", "confirmation_token", tok, "slot", "out")
	if err := f.tok.Consume("n4_set_slot", args, "p1", tok); err != nil {
		t.Fatal(err)
	}
	if got := BoundArgs(args).Keys(); strings.Join(got, ",") != "ord,slot" {
		t.Fatalf("bound args %v", got)
	}
	if args.Len() != 4 {
		t.Fatal("BoundArgs changed its input")
	}
}

func TestArgOrderDoesNotMatter(t *testing.T) {
	f := newFixture(t)
	tok, _, _ := f.tok.Issue("n4_set_slot", objOf("a", 1, "b", 2), "p1")
	if err := f.tok.Consume("n4_set_slot", objOf("b", 2, "a", 1), "p1", tok); err != nil {
		t.Fatal(err)
	}
}

func TestTokenFromAnotherProcessKeyIsRefused(t *testing.T) {
	f := newFixture(t)
	other := New(300, func() float64 { return f.now })
	tok, _, _ := other.Issue("n4_set_slot", f.args, "p1")
	refused(t, f.consume(tok), mismatch)
}

func TestExpiredEntriesArePrunedOnIssue(t *testing.T) {
	f := newFixture(t)
	f.issue()
	f.now = 1300 // pruned when not after now: an entry expiring exactly now goes
	f.issue()
	if len(f.tok.pending) != 1 {
		t.Fatalf("pending %v", f.tok.pending)
	}
	f.now = 1599.5
	f.issue()
	if len(f.tok.pending) != 2 {
		t.Fatalf("an entry still live was pruned: %v", f.tok.pending)
	}
}

func TestMismatchIsCheckedBeforeExpiryAndExpiryBeforeReuse(t *testing.T) {
	f := newFixture(t)
	tok := f.issue()
	f.now = 5000
	refused(t, f.tok.Consume("n4_other", f.args, "p1", tok), mismatch)
	refused(t, f.consume(tok), expired)
}

func TestRedactionMasksSecretLookingKeysAtAnyDepth(t *testing.T) {
	args := objOf("ord", "station:|slot:/A", "Password", "p1", "api_secret", "s1", "confirmation_token", "t1",
		"my_Credential", "c1", "nested", objOf("passphrase", "p2", "ok", 1), "items", []any{objOf("token", "t2", "n", 2)},
		"TO\u212aEN", "k1", "credent\u0130al", "kept", "passes", []any{"x"})
	red, _ := pyjson.Marshal(Redact(args), pyjson.Options{})
	want := `{"ord": "station:|slot:/A", "Password": "***", "api_secret": "***", "confirmation_token": "***", ` +
		`"my_Credential": "***", "nested": {"passphrase": "***", "ok": 1}, "items": [{"token": "***", "n": 2}], ` +
		`"TO\u212aEN": "***", "credent\u0130al": "kept", "passes": "***"}`
	if string(red) != want {
		t.Fatalf("got  %s\nwant %s", red, want)
	}
}

func TestRedactionDoesNotMutateTheInput(t *testing.T) {
	args := objOf("password", "p1", "list", []any{objOf("token", "t")})
	Redact(args)
	got, _ := pyjson.Marshal(args, pyjson.Options{})
	if string(got) != `{"password": "p1", "list": [{"token": "t"}]}` {
		t.Fatalf("input changed: %s", got)
	}
}
