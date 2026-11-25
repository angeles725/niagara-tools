package n4probe

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// station is a fake point: a press at /press turns out.value true after delay and, when hold is false,
// back to false after delay+length; hold keeps it true for good.
type station struct {
	mu                sync.Mutex
	pressed           time.Time
	delay, length     time.Duration
	hold              bool
	pressBody, pressT string
	auth              authLog
	pressStatus       int
}

func (s *station) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/press", func(w http.ResponseWriter, r *http.Request) {
		s.auth.note(r)
		b, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.pressed, s.pressBody, s.pressT = time.Now(), string(b), r.Method+" "+r.Header.Get("Content-Type")
		s.mu.Unlock()
		if s.pressStatus != 0 {
			w.WriteHeader(s.pressStatus)
		}
	})
	mux.HandleFunc("/obs", func(w http.ResponseWriter, r *http.Request) {
		s.auth.note(r)
		s.mu.Lock()
		since := time.Since(s.pressed)
		on := !s.pressed.IsZero() && since >= s.delay && (s.hold || since < s.delay+s.length)
		s.mu.Unlock()
		fmt.Fprintf(w, `{"out":{"value":%t},"ledger":{"n":7},"list":[{"v":"a"}]}`, on)
	})
	return mux
}

func pulseArgs(t *testing.T, base string, extra ...string) []string {
	return append([]string{"pulse", "--press", base + "/press", "--observe", base + "/obs", "--env-file", writeEnv(t, creds),
		"--rate", "200", "--allow-rate-above-4"}, extra...)
}

var changeLine = regexp.MustCompile(`(?m)^\s*(\d+) ms  (\S+)  (\S+) -> (\S+)$`)

func TestPulseTimelineAndRelease(t *testing.T) {
	st := &station{delay: 60 * time.Millisecond, length: 150 * time.Millisecond}
	srv := httptest.NewServer(st.handler())
	defer srv.Close()
	code, out, errs := run(pulseArgs(t, srv.URL, "--field", "out.value", "--duration", "5s")...)
	if code != ExitOK || errs != "" {
		t.Fatalf("code %d err %q\n%s", code, errs, out)
	}
	m := changeLine.FindAllStringSubmatch(out, -1)
	if len(m) != 2 || m[0][2] != "out.value" || m[0][3] != "false" || m[0][4] != "true" || m[1][3] != "true" || m[1][4] != "false" {
		t.Fatalf("timeline lines = %v, want false -> true then true -> false:\n%s", m, out)
	}
	rise, _ := strconv.Atoi(m[0][1])
	fall, _ := strconv.Atoi(m[1][1])
	if rise < 50 || rise > 400 || fall-rise < 120 || fall-rise > 600 {
		t.Errorf("rise at %d ms and fall at %d ms, want about 60 and 210 ms", rise, fall)
	}
	for _, want := range []string{"press status 200", "baseline out.value=false", "released after " + m[1][1] + " ms"} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.pressBody != "{}" || st.pressT != "POST application/json" {
		t.Errorf("press sent %q as %q, want {} as POST application/json", st.pressBody, st.pressT)
	}
	if st.auth.noAut != 0 || st.auth.pass != secret || strings.Contains(out, secret) {
		t.Errorf("auth: %d unauthenticated requests; secret printed: %v", st.auth.noAut, strings.Contains(out, secret))
	}
}

func TestPulseWatchesSeveralFieldsAndSendsTheBody(t *testing.T) {
	st := &station{delay: 30 * time.Millisecond, length: 80 * time.Millisecond}
	srv := httptest.NewServer(st.handler())
	defer srv.Close()
	code, out, _ := run(pulseArgs(t, srv.URL, "--field", "out.value", "--field", "ledger.n", "--field", "list.0.v", "--press-body", `{"p":1}`, "--duration", "1s")...)
	// ledger.n and list.0.v never change, so the pulse is not released: the unchanged fields are named.
	if code != ExitNotReleased || !strings.Contains(out, "never changed: ledger.n, list.0.v") {
		t.Errorf("code %d, want 3 naming the unchanged fields:\n%s", code, out)
	}
	if !strings.Contains(out, "baseline ledger.n=7") || !strings.Contains(out, `list.0.v="a"`) {
		t.Errorf("baselines missing:\n%s", out)
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.pressBody != `{"p":1}` {
		t.Errorf("press body %q, want the --press-body", st.pressBody)
	}
}

func TestPulseNotReleasedExitsThree(t *testing.T) {
	st := &station{delay: 20 * time.Millisecond, hold: true}
	srv := httptest.NewServer(st.handler())
	defer srv.Close()
	start := time.Now()
	code, out, _ := run(pulseArgs(t, srv.URL, "--field", "out.value", "--duration", "400ms")...)
	if code != ExitNotReleased || !strings.Contains(out, "not released within 400ms") || !strings.Contains(out, "not back at baseline: out.value") {
		t.Errorf("code %d, want 3 with the reason:\n%s", code, out)
	}
	if !strings.Contains(out, "false -> true") || strings.Contains(out, "released after") {
		t.Errorf("want the rise only:\n%s", out)
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Errorf("run took %s, the duration is 400ms", el)
	}
}

func TestPulsePressTransportFailureExitsOne(t *testing.T) {
	st := &station{}
	srv := httptest.NewServer(st.handler())
	args := pulseArgs(t, srv.URL, "--field", "out.value", "--duration", "300ms")
	gone := httptest.NewServer(http.NotFoundHandler())
	args[2] = gone.URL + "/press" // the press goes to a closed port, the observe URL is up
	gone.Close()
	code, out, errs := run(args...)
	srv.Close()
	if code != ExitFailed || !strings.Contains(errs, "press failed") || strings.Contains(out, "released") {
		t.Errorf("code %d err %q out %q, want 1 and press failed", code, errs, out)
	}
}

func TestPulseUsageErrors(t *testing.T) {
	base := "http://127.0.0.1:1"
	env := writeEnv(t, creds)
	for name, args := range map[string][]string{
		"no field":   {"pulse", "--press", base + "/p", "--observe", base + "/o", "--env-file", env},
		"no observe": {"pulse", "--press", base + "/p", "--field", "a", "--env-file", env},
		"no press":   {"pulse", "--observe", base + "/o", "--field", "a", "--env-file", env},
		"bad body":   {"pulse", "--press", base + "/p", "--observe", base + "/o", "--field", "a", "--env-file", env, "--press-body", "{nope"},
		"rate cap":   {"pulse", "--press", base + "/p", "--observe", base + "/o", "--field", "a", "--env-file", env, "--rate", "9"},
	} {
		if code, _, errs := run(args...); code != ExitUsage || errs == "" {
			t.Errorf("%s: code %d err %q, want a usage error", name, code, errs)
		}
	}
}

func TestFieldPath(t *testing.T) {
	doc, err := decode([]byte(`{"a":{"b":[{"c":1.50},true]},"s":"x","n":null}`))
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{"a.b.0.c": "1.50", "a.b.1": "true", "s": `"x"`, "n": "null", "a.x": absent, "a.b.9": absent, "s.q": absent} {
		if got := render(doc, path); got != want {
			t.Errorf("%s = %s, want %s", path, got, want)
		}
	}
}
