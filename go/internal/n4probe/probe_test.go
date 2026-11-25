package n4probe

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const secret = "s3cret pw"

// creds is an env file with the two probe keys; the secret has a space so it is quoted.
const creds = "# probe\nexport N4_PROBE_USER=probe\nN4_PROBE_PASSWORD='" + secret + "'\n"

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "probe.env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// run is one n4-probe run; the process environment carries decoy credentials that must never be used.
func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string { return "decoy-" + name }}
	code = Main(args, s)
	return code, o.String(), e.String()
}

// authLog records the basic-auth pair of every request a fake station got.
type authLog struct {
	mu          sync.Mutex
	user, pass  string
	seen, noAut int
}

func (a *authLog) note(r *http.Request) {
	u, p, ok := r.BasicAuth()
	a.mu.Lock()
	defer a.mu.Unlock()
	a.seen++
	if !ok {
		a.noAut++
	}
	a.user, a.pass = u, p
}

func TestRateCap(t *testing.T) {
	env := writeEnv(t, creds)
	for _, tc := range []struct {
		rate, want string
		code       int
	}{
		{"5", "above the 4 Hz cap", ExitUsage},
		{"0", "--rate must be greater than 0", ExitUsage},
		{"-1", "--rate must be greater than 0", ExitUsage},
	} {
		code, out, errs := run("stall", "--url", "http://127.0.0.1:1/x", "--env-file", env, "--rate", tc.rate)
		if code != tc.code || out != "" || !strings.Contains(errs, tc.want) {
			t.Errorf("--rate %s: code %d out %q err %q, want %d and %q", tc.rate, code, out, errs, tc.code, tc.want)
		}
	}
}

func TestRateAboveCapNeedsTheFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	env := writeEnv(t, creds)
	code, _, errs := run("stall", "--url", srv.URL, "--env-file", env, "--rate", "50", "--allow-rate-above-4", "--duration", "100ms")
	if code != ExitOK {
		t.Errorf("--allow-rate-above-4: code %d err %q, want 0", code, errs)
	}
}

func TestDurationCap(t *testing.T) {
	env := writeEnv(t, creds)
	for _, d := range []string{"10m1s", "11m", "0s", "-5s", "soon"} {
		code, out, errs := run("stall", "--url", "http://127.0.0.1:1/x", "--env-file", env, "--duration", d)
		if code != ExitUsage || out != "" || errs == "" {
			t.Errorf("--duration %s: code %d out %q err %q, want usage error", d, code, out, errs)
		}
	}
}

func TestEnvFileRequiredAndNeverTheProcessEnvironment(t *testing.T) {
	url := "http://127.0.0.1:1/x"
	for name, args := range map[string][]string{
		"no flag":       {"stall", "--url", url},
		"missing file":  {"stall", "--url", url, "--env-file", filepath.Join(t.TempDir(), "nope.env")},
		"no password":   {"stall", "--url", url, "--env-file", writeEnv(t, "N4_PROBE_USER=probe\n")},
		"shell command": {"stall", "--url", url, "--env-file", writeEnv(t, "N4_PROBE_USER=probe\nN4_PROBE_PASSWORD=$(echo hunter2)\n")},
	} {
		code, out, errs := run(args...)
		if code != ExitUsage || out != "" || errs == "" {
			t.Errorf("%s: code %d out %q err %q, want usage error", name, code, out, errs)
		}
		if strings.Contains(errs, "decoy") || strings.Contains(errs, "hunter2") {
			t.Errorf("%s: error leaks a value: %q", name, errs)
		}
	}
	if code, _, errs := run("stall", "--url", url, "--env-file", writeEnv(t, creds), "--password", "x"); code != ExitUsage || !strings.Contains(errs, "password") {
		t.Errorf("a password flag must not exist: code %d err %q", code, errs)
	}
}

func TestCredentialsAreSentAsBasicAuthAndNeverPrinted(t *testing.T) {
	var log authLog
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { log.note(r) }))
	defer srv.Close()
	code, out, errs := run("stall", "--url", srv.URL+"/x?token=qq", "--env-file", writeEnv(t, creds), "--rate", "100", "--allow-rate-above-4", "--duration", "150ms")
	if code != ExitOK {
		t.Fatalf("code %d err %q", code, errs)
	}
	if log.seen == 0 || log.noAut != 0 || log.user != "probe" || log.pass != secret {
		t.Errorf("station saw %d requests, %d without auth, user %q: want basic auth probe/<password>", log.seen, log.noAut, log.user)
	}
	for _, text := range []string{out, errs} {
		if strings.Contains(text, secret) || strings.Contains(text, "probe:") || strings.Contains(text, "token=qq") {
			t.Errorf("output leaks a credential or the query: %q", text)
		}
	}
}

func TestURLWithCredentialsIsRefusedWithoutEchoing(t *testing.T) {
	code, _, errs := run("stall", "--url", "http://bob:hunter2@127.0.0.1:1/x", "--env-file", writeEnv(t, creds))
	if code != ExitUsage || strings.Contains(errs, "hunter2") || !strings.Contains(errs, "--url must not carry credentials") {
		t.Errorf("code %d err %q", code, errs)
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"bogus"}, {"stall"}, {"stall", "--nope"}, {"stall", "--url", "ftp://h/x"}, {"pulse"}} {
		if code, _, errs := run(args...); code != ExitUsage || errs == "" {
			t.Errorf("%v: code %d err %q, want a usage error", args, code, errs)
		}
	}
	if code, out, _ := run("--help"); code != ExitOK || !strings.Contains(out, "n4call") {
		t.Errorf("--help: code %d out %q, want the usage text that points to n4call", code, out)
	}
}

func TestInsecureTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	env := writeEnv(t, creds)
	args := []string{"stall", "--url", srv.URL, "--env-file", env, "--rate", "50", "--allow-rate-above-4", "--duration", "100ms"}
	if code, _, _ := run(args...); code != ExitFailed {
		t.Errorf("self-signed station without --insecure-tls: code %d, want %d", code, ExitFailed)
	}
	if code, _, errs := run(append(args, "--insecure-tls")...); code != ExitOK {
		t.Errorf("with --insecure-tls: code %d err %q, want 0", code, errs)
	}
}
