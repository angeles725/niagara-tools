package obixlinkaudit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// golden is one run of the bash tool, recorded from tests/fixtures/obix-link-audit with OBIX_USER and
// OBIX_PASS unset.
type golden struct {
	Name           string
	Args           []string
	Exit           int
	Stdout, Stderr string
}

// run calls Main with env as the whole environment.
func run(env map[string]string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string { return env[k] }})
	return code, o.String(), e.String()
}

// declared are the goldens where the port differs from bash on purpose: bash grepped the raw text for
// display="Indirect|Direct: ...", the port reads the XML. A malformed response is exit 3 instead of a
// partial link set (rows MISSING or even MATCH from a broken file), and a link written with &#8594;,
// -&gt;, single quotes or spaces around = is a link instead of a false MISSING.
var declared = map[string]golden{
	"truncated": {Exit: ExitEnv, Stderr: "obix-link-audit: truncated.xml:5:51: unexpected EOF\n"},
	"badclose":  {Exit: ExitEnv, Stderr: "obix-link-audit: bad-close.xml:1:84: element <obj> opened at 1:1 closed by </obk>\n"},
	"entities": {Exit: ExitOK, Stdout: "MATCH  obix-link-audit  comp1State  linked from declared comp1Running in Rack\n" +
		"MATCH  obix-link-audit  comp2State  linked from declared comp2Running\n" +
		"MATCH  obix-link-audit  comp3State  linked from declared comp3Running in Rack\n" +
		"MATCH  obix-link-audit  inDrip  linked from declared dripActive\n" +
		"SKIP  obix-link-audit  freezeActive  source not declared in the wiring map (fill it before commissioning)\n" +
		"obix-link-audit: 4 MATCH · 0 MISMATCH · 0 MISSING · 1 SKIP  ->  CLEAN\n"},
}

func TestBashGoldens(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "bash-goldens.json"))
	if err != nil {
		t.Fatal(err)
	}
	var goldens []golden
	if err := json.Unmarshal(data, &goldens); err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join("..", "..", "..", "tests", "fixtures", "obix-link-audit"))
	if len(goldens) < 30 {
		t.Fatalf("%d goldens, want the recorded set", len(goldens))
	}
	for _, g := range goldens {
		want, differs := declared[g.Name]
		if !differs {
			want = g
		} else if want.Exit == g.Exit && want.Stdout == g.Stdout && want.Stderr == g.Stderr {
			t.Errorf("%s: declared difference matches bash; drop it from declared", g.Name)
		}
		code, out, errs := run(nil, g.Args...)
		if code != want.Exit || out != want.Stdout || errs != want.Stderr {
			t.Errorf("%s %q:\n got %d\n%s%s\nwant %d\n%s%s", g.Name, g.Args, code, out, errs, want.Exit, want.Stdout, want.Stderr)
		}
	}
}

// TestLive runs the live mode against local httptest stations only.
func TestLive(t *testing.T) {
	t.Parallel()
	fx := filepath.Join("..", "..", "..", "tests", "fixtures", "obix-link-audit")
	facade, err := os.ReadFile(filepath.Join(fx, "facade.xml"))
	if err != nil {
		t.Fatal(err)
	}
	creds := map[string]string{"OBIX_USER": "reader", "OBIX_PASS": "secret"}
	const warn = "obix-link-audit: WARNING: http:// base sends the oBIX credentials in clear text; use https:// (--insecure for a self-signed certificate)\n"
	wiring := filepath.Join(fx, "wiring-map.md")
	for _, tc := range []struct {
		name      string
		tls       bool
		status    int
		body      string
		env       map[string]string
		base      string // appended to the station URL
		comp      string
		extra     []string
		code      int
		requests  int
		errPrefix string // stderr starts with it
		errSuffix string // and ends with it
	}{
		{"one GET, http warns", false, 200, string(facade), creds, "/obix", "/Plant/Facade", nil, ExitIssues, 1, warn, warn},
		{"slashes trimmed once", false, 200, string(facade), creds, "/obix/", "Plant/Facade/", nil, ExitIssues, 1, warn, warn},
		{"no credentials, no request", false, 200, "", map[string]string{"OBIX_USER": "reader"}, "/obix", "/Plant/Facade", nil, ExitEnv, 0,
			"obix-link-audit: live mode needs OBIX_USER and OBIX_PASS (a read-only oBIX user)\n", ""},
		{"404 is GET failed", false, 404, "", creds, "/obix", "/Plant/Facade", nil, ExitEnv, 1, warn + "obix-link-audit: the station answered HTTP 404 Not Found\n",
			"obix-link-audit: GET failed: {URL}/obix/config/Plant/Facade/\n"},
		{"empty body is exit 3", false, 200, "", creds, "/obix", "/Plant/Facade", nil, ExitEnv, 1, warn + "obix-link-audit: {URL}/obix/config/Plant/Facade/:", ""},
		{"self-signed refused", true, 200, string(facade), creds, "/obix", "/Plant/Facade", nil, ExitEnv, 0, "obix-link-audit: Get ",
			"obix-link-audit: GET failed: {URL}/obix/config/Plant/Facade/\n"},
		{"--insecure accepts it, https no warning", true, 200, string(facade), creds, "/obix", "/Plant/Facade", []string{"--insecure"}, ExitIssues, 1, "", ""},
	} {
		s := newStation(t, tc.tls, tc.status, tc.body)
		args := append([]string{"--map", wiring, "--obix", s.URL + tc.base, "--component", tc.comp}, tc.extra...)
		code, out, errs := run(tc.env, args...)
		pre, suf := strings.ReplaceAll(tc.errPrefix, "{URL}", s.URL), strings.ReplaceAll(tc.errSuffix, "{URL}", s.URL)
		if code != tc.code || !strings.HasPrefix(errs, pre) || !strings.HasSuffix(errs, suf) || (pre == "" && errs != "") {
			t.Errorf("%s: exit %d stderr %q, want %d %q...%q", tc.name, code, errs, tc.code, pre, suf)
		}
		if strings.Contains(out+errs, "secret") {
			t.Errorf("%s: the password reached the output", tc.name)
		}
		if tc.code == ExitIssues && !strings.Contains(out, "MISMATCH  obix-link-audit  comp1State  declared comp1Running in Rack, linked from relayOut1") {
			t.Errorf("%s: stdout %q", tc.name, out)
		}
		got := s.requests()
		if len(got) != tc.requests || (len(got) == 1 && got[0] != "GET /obix/config/Plant/Facade/ reader:secret") {
			t.Errorf("%s: requests %q, want %d GET of /obix/config/Plant/Facade/", tc.name, got, tc.requests)
		}
	}
}

// TestLiveArgumentGuards: a base or component that cannot name one station GET is exit 3 before any
// request, and no password given in the URL reaches the output.
func TestLiveArgumentGuards(t *testing.T) {
	t.Parallel()
	creds := map[string]string{"OBIX_USER": "reader", "OBIX_PASS": "secret"}
	wiring := filepath.Join("..", "..", "..", "tests", "fixtures", "obix-link-audit", "wiring-map.md")
	s := newStation(t, false, 200, "<obj/>")
	host := strings.TrimPrefix(s.URL, "http://")
	for _, tc := range []struct{ name, base, comp, stderr string }{
		{"userinfo in the base", "http://admin:Pa55@" + host + "/obix", "/Plant/Facade",
			"obix-link-audit: --obix must not carry credentials (user:password@); pass them in OBIX_USER / OBIX_PASS\n"},
		{"user only in the base", "http://admin@" + host + "/obix", "/Plant/Facade",
			"obix-link-audit: --obix must not carry credentials (user:password@); pass them in OBIX_USER / OBIX_PASS\n"},
		{"no scheme", "station.example/obix", "/Plant/Facade",
			"obix-link-audit: --obix needs an http:// or https:// base with a host: station.example/obix\n"},
		{"ftp scheme", "ftp://" + host + "/obix", "/Plant/Facade",
			"obix-link-audit: --obix needs an http:// or https:// base with a host: ftp://" + host + "/obix\n"},
		{"no host", "https:///obix", "/Plant/Facade", "obix-link-audit: --obix needs an http:// or https:// base with a host: https:///obix\n"},
		{"query in the base", s.URL + "/obix?x=1", "/Plant/Facade",
			"obix-link-audit: --obix must not carry a query or fragment: " + s.URL + "/obix?x=1\n"},
		{"unparseable base", "http://[::1/obix", "/Plant/Facade", "obix-link-audit: --obix is not a URL\n"},
	} {
		code, out, errs := run(creds, "--map", wiring, "--obix", tc.base, "--component", tc.comp)
		if code != ExitEnv || out != "" || errs != tc.stderr {
			t.Errorf("%s: exit %d stdout %q stderr %q, want 3 %q", tc.name, code, out, errs, tc.stderr)
		}
		if strings.Contains(errs, "Pa55") {
			t.Errorf("%s: the URL password reached stderr", tc.name)
		}
	}
	if got := s.requests(); len(got) != 0 {
		t.Errorf("requests %q, want none", got)
	}
}
