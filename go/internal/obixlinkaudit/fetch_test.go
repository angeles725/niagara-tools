package obixlinkaudit

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// station is a local test server that answers every request with status and body and records each
// request as "METHOD path user:pass".
type station struct {
	*httptest.Server
	mu   sync.Mutex
	seen []string
}

func newStation(t *testing.T, tls bool, status int, body string) *station {
	t.Helper()
	s := &station{}
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := r.BasicAuth()
		s.mu.Lock()
		s.seen = append(s.seen, r.Method+" "+r.URL.Path+" "+u+":"+p)
		s.mu.Unlock()
		if status == http.StatusFound {
			w.Header().Set("Location", "/elsewhere/")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	s.Server = httptest.NewUnstartedServer(h)
	s.Config.ErrorLog = log.New(io.Discard, "", 0) // the refused self-signed handshake is expected
	if tls {
		s.StartTLS()
	} else {
		s.Start()
	}
	t.Cleanup(s.Close)
	return s
}

func (s *station) requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func TestGetOneReadOnlyGETWithBasicAuth(t *testing.T) {
	t.Parallel()
	s := newStation(t, false, http.StatusOK, "<obj/>")
	body, err := get(s.URL+"/obix/config/P/F/", "reader", `p"a\ss #1 x`, false)
	if err != nil || string(body) != "<obj/>" {
		t.Fatalf("get = %q, %v", body, err)
	}
	if got := s.requests(); len(got) != 1 || got[0] != `GET /obix/config/P/F/ reader:p"a\ss #1 x` {
		t.Errorf("requests = %q, want one GET with the raw credentials", got)
	}
}

func TestGetErrorsAndRedirects(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		status int
		want   string // error substring; "" = the body is returned
	}{
		{"404 fails like curl -f", http.StatusNotFound, "the station answered HTTP 404 Not Found"},
		{"500 fails", http.StatusInternalServerError, "HTTP 500"},
		{"302 is not followed and fails", http.StatusFound, "redirect not followed: HTTP 302 Found"},
		{"304 fails too", http.StatusNotModified, "redirect not followed: HTTP 304 Not Modified"},
		{"200 returns the body", http.StatusOK, ""},
	} {
		s := newStation(t, false, tc.status, "<obj/>")
		body, err := get(s.URL+"/obix/config/P/", "u", "secret", false)
		switch {
		case tc.want == "" && (err != nil || string(body) != "<obj/>"):
			t.Errorf("%s: get = %q, %v; want the body", tc.name, body, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%s: err = %v, want %q", tc.name, err, tc.want)
		case err != nil && strings.Contains(err.Error(), "secret"):
			t.Errorf("%s: error text carries the password: %v", tc.name, err)
		}
		if n := len(s.requests()); n != 1 {
			t.Errorf("%s: %d requests, want 1", tc.name, n)
		}
	}
}

func TestGetVerifiesTheCertificateUnlessInsecure(t *testing.T) {
	t.Parallel()
	s := newStation(t, true, http.StatusOK, "<obj/>")
	if _, err := get(s.URL+"/obix/config/P/", "u", "p", false); err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Errorf("self-signed station without --insecure: err = %v, want a certificate error", err)
	}
	if body, err := get(s.URL+"/obix/config/P/", "u", "p", true); err != nil || string(body) != "<obj/>" {
		t.Errorf("--insecure: get = %q, %v", body, err)
	}
}

func TestGetCapsTheBody(t *testing.T) {
	t.Parallel()
	// Mutation: OLA-bodycap -- an unbounded read lets a hostile or broken station exhaust memory.
	s := newStation(t, false, http.StatusOK, strings.Repeat("x", maxBody+1))
	if body, err := get(s.URL+"/obix/config/P/", "u", "p", false); err == nil || !strings.Contains(err.Error(), "response larger than 32 MiB") {
		t.Errorf("oversized body: %d bytes, err = %v; want the cap error", len(body), err)
	}
}
