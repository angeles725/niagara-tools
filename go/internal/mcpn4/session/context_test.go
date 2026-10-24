package session

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

type clock struct{ now float64 }

func newContext(cfg Config) (*Context, *clock) {
	c := &clock{1000}
	ctx := New(cfg)
	ctx.Clock = func() float64 { return c.now }
	return ctx, c
}

func encode(t *testing.T, o *pyjson.Object) string {
	t.Helper()
	b, err := pyjson.Marshal(o, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestScrubHidesTheSecretOnlyOnceOneIsSet(t *testing.T) {
	t.Parallel()
	ctx := New(Config{})
	if got := ctx.Scrub("a pw b"); got != "a pw b" {
		t.Errorf("Scrub without a secret = %q", got)
	}
	ctx.SetSecret("pw")
	if got := ctx.Scrub("pw a pw"); got != "*** a ***" {
		t.Errorf("Scrub = %q", got)
	}
}

type closer struct{ closed int }

func (c *closer) Close() { c.closed++ }

func TestTierReportPerTierAndOptIn(t *testing.T) {
	t.Parallel()
	ctx := New(Config{AllowTierB: []string{"S"}})
	for _, tc := range []struct {
		version, verr, want string
	}{
		{"4.14.0.162", "", `{"version": "4.14.0.162", "version_source": "/obix/about/", "tier": "A", "tier_writes": "allowed"}`},
		{"4.15.3.28", "", `{"version": "4.15.3.28", "version_source": "/obix/about/", "tier": "B", "tier_writes": "allowed-by-opt-in"}`},
		{"", "/obix/about/ has no productVersion", `{"version": null, "version_source": null, "tier": "C", "tier_writes": "refused", "version_error": "/obix/about/ has no productVersion"}`},
	} {
		out := pyjson.NewObject()
		ctx.AddTierReport(out, NewSession(&closer{}, "S", "https://h", "2", tc.version, tc.verr))
		if got := encode(t, out); got != tc.want {
			t.Errorf("got  %s\nwant %s", got, tc.want)
		}
	}
	out := pyjson.NewObject()
	ctx.AddTierReport(out, nil)
	if got, want := encode(t, out), `{"version": null, "version_source": null, "tier": null, "tier_writes": null}`; got != want {
		t.Errorf("no session: %s", got)
	}
}

func TestCloseClearsTheSessionAndClosesItsClient(t *testing.T) {
	t.Parallel()
	ctx := New(Config{})
	ctx.Close() // no session: nothing to do
	c := &closer{}
	ctx.Session = NewSession(c, "S", "https://h", "2", "", "")
	if !ctx.Connected() {
		t.Fatal("not connected with a session")
	}
	ctx.Close()
	if ctx.Connected() || c.closed != 1 {
		t.Errorf("connected=%v closed=%d", ctx.Connected(), c.closed)
	}
	a, b := NewSession(c, "S", "", nil, "", ""), NewSession(c, "S", "", nil, "", "")
	if len(a.SessionID) != 32 || a.SessionID == b.SessionID || !a.IdentityVerified {
		t.Errorf("session ids %q %q", a.SessionID, b.SessionID)
	}
}

func TestModeAndStationNames(t *testing.T) {
	t.Parallel()
	ctx := New(Config{Stations: map[string]string{"b": "https://x", "a": "https://y", "é": "https://z"}})
	if got := strings.Join(ctx.StationNames(), ","); got != "a,b,é" || ctx.Mode() != "read-only" {
		t.Errorf("names %s, mode %s", got, ctx.Mode())
	}
	if New(Config{AllowWrites: true}).Mode() != "writes-allowed" {
		t.Error("write mode")
	}
}
