package tiers_test

// Go twins of the test_tiers.TestConnectReportsTier tests that patch box.BoxClient.about or build
// a Server in-process (go_server.go_twin("tiers", ...)).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func field(o *pyjson.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

func TestAnAboutAuthRejectionIsReportedNotSwallowed(t *testing.T) {
	st := readtoolstest.NewStation()
	st.AboutErr = &box.Error{Kind: box.KindAuth, Msg: box.AuthMessage(401), Channel: "obix", Key: "about"}
	s := readtoolstest.NewServer(st, session.Config{})
	out := s.Connect(t)
	if field(out, "version") != nil || field(out, "tier") != "C" || field(out, "tier_writes") != "refused" {
		t.Errorf("connect %s", pyjson.Repr(out))
	}
	want := "/obix/about/ refused: " + box.AuthMessage(401) + "; the station may count it as a failed " +
		"login, so check that this user may read oBIX before reconnecting"
	if got := field(out, "version_error"); got != want {
		t.Errorf("version_error %q\nwant %q", got, want)
	}
	if s.Ctx.Session == nil { // detection never fails the connect
		t.Fatal("no session")
	}
	if _, _, desc := s.Call(t, "n4_describe_session"); field(desc, "version_error") != want {
		t.Errorf("describe %s", pyjson.Repr(desc))
	}
}

func TestATransportFailureOrAClientWithoutAboutIsTierC(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(*readtoolstest.Station)
		want string
	}{
		{"transport", func(st *readtoolstest.Station) {
			st.AboutErr = &box.Error{Kind: box.KindBox, Msg: "transport failure: reset", Channel: "obix", Key: "about"}
		}, "/obix/about/ unreadable: transport failure: reset"},
		{"escaped", func(st *readtoolstest.Station) { st.AboutErr = readtoolstest.Escaped("TypeError") },
			"/obix/about/ unreadable (TypeError)"},
		{"no about", func(st *readtoolstest.Station) { st.NoAbout = true }, "this client cannot read /obix/about/"},
		{"no version", func(st *readtoolstest.Station) { st.Version = "" }, "/obix/about/ unreadable: HTTP 404 from station"},
	} {
		st := readtoolstest.NewStation()
		tc.set(st)
		out := readtoolstest.NewServer(st, session.Config{}).Connect(t)
		if field(out, "tier") != "C" || field(out, "version") != nil || field(out, "version_error") != tc.want {
			t.Errorf("%s: %s", tc.name, pyjson.Repr(out))
		}
	}
}

func TestOptInForAnUnconfiguredStationIsAStartError(t *testing.T) {
	for flag, set := range map[string]func(*cli.Config){
		"--allow-tier-b": func(c *cli.Config) { c.AllowTierB = []string{"ghost"} },
		"--allow-tier-c": func(c *cli.Config) { c.AllowTierC = []string{"ghost"} },
	} {
		cfg := cli.Defaults()
		cfg.AllowHTTP, cfg.Stations = true, []string{"FakeStation=http://h"}
		set(&cfg)
		if err := cli.Check(cfg); err == nil || !strings.HasPrefix(err.Error(), flag+" names unconfigured station(s): ghost") {
			t.Errorf("%s: Check = %v", flag, err)
		}
	}
}
