package cli

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

func TestCliParsesTheOptInFlags(t *testing.T) {
	t.Parallel()
	cfg, _, err := Parse([]string{"--allow-tier-b", "S1", "--allow-tier-b", "S2", "--allow-tier-c", "S3"})
	if err != nil || !reflect.DeepEqual(cfg.AllowTierB, []string{"S1", "S2"}) || !reflect.DeepEqual(cfg.AllowTierC, []string{"S3"}) {
		t.Errorf("Parse = %+v, %v", cfg, err)
	}
}

// TestLoadWaitAndHttpTimeoutFlagsReachTheClient: the operator timing flags (audit 2026-10-03 F11)
// and the station policy go from the command line through NewContext to the client n4_connect makes.
func TestLoadWaitAndHttpTimeoutFlagsReachTheClient(t *testing.T) {
	t.Parallel()
	cfg, _, uerr := Parse([]string{"--load-wait", "12.5", "--http-timeout", "90", "--allow-http-for-tests",
		"--station", "FakeStation=http://a/", "--station", "FakeStation=http://b/", "--credential-env", "LAB",
		"--insecure-tls", "FakeStation", "--allow-tier-c", "FakeStation", "--auth-cooldown", "45"})
	if uerr != nil || Check(cfg) != nil {
		t.Fatalf("start refused: %v", uerr)
	}
	ctx := NewContext(cfg)
	if ctx.HTTPTimeout != 90 || ctx.LoadWait != 12.5 || ctx.AuthCooldown != 45 || ctx.Mode() != "read-only" ||
		!ctx.TierOptIn["C"]["FakeStation"] || ctx.TierOptIn["B"]["FakeStation"] {
		t.Errorf("context %+v", ctx.Config)
	}
	st := readtoolstest.NewStation()
	env := map[string]string{"LAB_USER": "admin", "LAB_PASSWORD": readtoolstest.Password}
	ctx.Getenv = func(k string) string { return env[k] }
	args := pyjson.NewObject()
	args.Set("station", "FakeStation")
	if _, err := (&readtools.Tools{Ctx: ctx, NewClient: st.Factory}).Connect(args); err != nil {
		t.Fatal(err)
	}
	if c := st.Configs[0]; c.BaseURL != "http://b/" || c.Timeout != 90 || c.LoadWait != 12.5 || !c.InsecureTLS || !c.AllowHTTP {
		t.Errorf("client config %+v", c)
	}
}
