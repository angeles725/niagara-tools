package box

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// seen is one request a scriptedStation received.
type seen struct {
	Method, Path string
	Header       http.Header
	Body         []byte
	Frame        *pyjson.Object // the decoded BOX frame of a POST, else nil
}

func (s seen) key() string {
	if s.Frame == nil {
		return ""
	}
	m, _ := s.Frame.Get("m")
	msgs, _ := m.([]any)
	if len(msgs) == 0 {
		return ""
	}
	k, _ := get(msgs[0], "k")
	if k == "callssc" {
		b, _ := get(msgs[0], "b")
		sck, _ := get(b, "sck")
		return "callssc:" + pyjson.Str(sck)
	}
	return pyjson.Str(k)
}

type reply struct {
	Status int
	Header map[string]string
	Body   string
	Hang   bool
	Raw    string
}

type scriptedStation struct {
	*httptest.Server
	mu     sync.Mutex
	seen   []seen
	answer func(s seen) *reply // nil result: the default answer
}

func newScripted(t *testing.T, answer func(s seen) *reply) *scriptedStation {
	st := &scriptedStation{answer: answer}
	st.Server = httptest.NewServer(http.HandlerFunc(st.serve))
	t.Cleanup(st.Close)
	return st
}

func (st *scriptedStation) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s := seen{Method: r.Method, Path: r.URL.EscapedPath(), Header: r.Header.Clone(), Body: body}
	if r.Method == http.MethodPost {
		if v, err := pyjson.Unmarshal(body); err == nil {
			s.Frame, _ = v.(*pyjson.Object)
		}
	}
	st.mu.Lock()
	st.seen = append(st.seen, s)
	st.mu.Unlock()
	var rep *reply
	if st.answer != nil {
		rep = st.answer(s)
	}
	if rep == nil {
		rep = defaultAnswer(s)
	}
	if rep.Hang || rep.Raw != "" {
		conn, _, _ := w.(http.Hijacker).Hijack()
		io.WriteString(conn, rep.Raw)
		conn.Close()
		return
	}
	for k, v := range rep.Header {
		w.Header().Set(k, v)
	}
	w.WriteHeader(rep.Status)
	io.WriteString(w, rep.Body)
}

func (st *scriptedStation) requests() []seen {
	st.mu.Lock()
	defer st.mu.Unlock()
	return append([]seen(nil), st.seen...)
}

func (st *scriptedStation) keys() []string {
	var out []string
	for _, s := range st.requests() {
		out = append(out, s.key())
	}
	return out
}

func boxReply(s seen, b any, isErr bool) *reply {
	n, _ := s.Frame.Get("n")
	t := "rp"
	if isErr {
		t = "e"
	}
	out, _ := pyjson.Marshal(obj("v", "2.3", "p", "box", "n", n, "m", []any{obj("r", 0, "t", t, "b", b)}),
		pyjson.Options{})
	return &reply{Status: 200, Header: map[string]string{"Content-Type": "application/json"}, Body: string(out)}
}

func defaultAnswer(s seen) *reply {
	if s.Frame == nil {
		return &reply{Status: 404}
	}
	switch s.key() {
	case "make":
		return boxReply(s, "sess-1", false)
	case "makessc":
		return boxReply(s, obj("isReadonly", false), false)
	case "callssc:loadRoot":
		return boxReply(s, obj("h", "2", "t", "baja:Station"), false)
	case "del", "callssc:loadSlots":
		return boxReply(s, nil, false)
	case "pollchgs":
		return boxReply(s, []any{}, false)
	}
	return boxReply(s, obj("m", "unknown key"), true)
}

func newClient(t *testing.T, url string, mod func(*Config)) *Client {
	t.Helper()
	cfg := Config{BaseURL: url, Username: "admin", Password: "secret", AllowHTTP: true}
	if mod != nil {
		mod(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	c.Sleep = func(float64) {}
	return c
}

func boxErr(t *testing.T, err error, kind Kind) *Error {
	t.Helper()
	e, ok := err.(*Error)
	if !ok || e.Kind != kind {
		t.Fatalf("err = %#v, want *Error of kind %v", err, kind)
	}
	return e
}

func TestConstants(t *testing.T) {
	if SessionComponentID != "cs1" || DefaultRootHandle != "2" || MaxPendingEvents != 500 ||
		MaxReplyBytes != 32*1024*1024 || ProtocolVersion != "2.3" || Channel != "ssession" {
		t.Fatal("a box.py constant drifted")
	}
}

func TestHTTPRefusedWithoutAllowHTTP(t *testing.T) {
	_, err := New(Config{BaseURL: "http://127.0.0.1:1", Username: "u", Password: "p"})
	if ve, ok := err.(*pyjson.ValueError); !ok || ve.Msg != "refusing http:// (password would travel in clear); use https" {
		t.Fatalf("err = %v", err)
	}
	_, err = New(Config{BaseURL: "ftp://x", Username: "u", Password: "p"})
	if ve, ok := err.(*pyjson.ValueError); !ok || ve.Msg != "base_url must start with https://" {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPSAcceptedWithoutFlag(t *testing.T) {
	c, err := New(Config{BaseURL: "https://station.example/", Username: "u", Password: "p"})
	if err != nil || c.BaseURL() != "https://station.example" {
		t.Fatalf("New = %v, %v", c, err)
	}
}

func TestReprHidesPassword(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:1", func(c *Config) { c.Password = "s3cr3t-pw" })
	if got := c.String(); got != "BoxClient(base_url='http://127.0.0.1:1', username='admin')" {
		t.Fatalf("String() = %q", got)
	}
}

func TestRedirectIsRefusedAndCredentialsNeverReachTarget(t *testing.T) {
	target := newScripted(t, nil)
	for _, code := range []int{301, 302, 303, 307, 308} {
		st := newScripted(t, func(s seen) *reply {
			return &reply{Status: code, Header: map[string]string{"Location": target.URL + "/box/"}}
		})
		_, err := newClient(t, st.URL, nil).Call(Channel, "make", obj())
		e := boxErr(t, err, KindBox)
		want := "refusing HTTP " + itoa(code) + " redirect to host 127.0.0.1 (credentials are never forwarded)"
		if e.Msg != want || e.Channel != "" || e.Key != "" {
			t.Fatalf("redirect %d: %+v", code, e)
		}
	}
	if len(target.requests()) != 0 {
		t.Fatalf("target saw %v", target.keys())
	}
}

func itoa(n int) string { return pyjson.Repr(n) }

func TestTransportFailuresAreBoxErrors(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close() // nothing listens there any more: connection refused
	_, err = newClient(t, "http://"+addr, nil).Call(Channel, "make", obj())
	if e := boxErr(t, err, KindBox); e.Msg != "transport failure: <urlopen error [Errno 111] Connection refused>" {
		t.Fatalf("refused: %q", e.Msg)
	}
	st := newScripted(t, func(s seen) *reply { return &reply{Hang: true} })
	_, err = newClient(t, st.URL, nil).Call(Channel, "make", obj())
	if e := boxErr(t, err, KindBox); e.Msg != "transport failure: Remote end closed connection without response" {
		t.Fatalf("hang: %q", e.Msg)
	}
}

func TestReplyShapes(t *testing.T) {
	big := `{"pad": "` + strings.Repeat("x", 4096) + `"}`
	for _, tt := range []struct {
		name, body, want string
	}{
		{"invalid json", "<html>not json", "transport failure: Expecting value: line 1 column 1 (char 0)"},
		{"not utf-8", "\xff", "transport failure: 'utf-8' codec can't decode byte 0xff in position 0: invalid start byte"},
		{"oversized", big, "BOX reply larger than 1024 bytes: refused, not parsed in part"},
		{"missing m", `{"n": 1}`, "malformed reply: missing or empty message list"},
		{"empty m", `{"n": 1, "m": []}`, "malformed reply: missing or empty message list"},
		{"m not a list", `{"n": 1, "m": "x"}`, "malformed reply: missing or empty message list"},
		{"non-dict message", `{"n": 1, "m": ["x"]}`, "malformed reply: missing or empty message list"},
		{"reply not a dict", `[1, 2]`, "malformed reply: not an object"},
		{"error with non-dict b", `{"n": 1, "m": [{"t": "e", "b": "boom"}]}`, "station error"},
		{"error with null b", `{"n": 1, "m": [{"t": "e", "b": null}]}`, "station error"},
		{"error message", `{"n": 1, "m": [{"t": "e", "b": {"m": "boom"}}]}`, "boom"},
		{"seq mismatch", `{"n": 42, "m": [{"t": "rp", "b": "sess-1"}]}`, "reply seq 42 does not match request seq 1"},
		{"seq as a string", `{"n": "1", "m": [{"t": "rp", "b": "sess-1"}]}`, "reply seq '1' does not match request seq 1"},
	} {
		st := newScripted(t, func(s seen) *reply { return &reply{Status: 200, Body: tt.body} })
		c := newClient(t, st.URL, func(c *Config) { c.MaxReplyBytes = 1024 })
		_, err := c.Call(Channel, "make", obj())
		if e := boxErr(t, err, KindBox); e.Msg != tt.want || e.Channel != "ssession" || e.Key != "make" {
			t.Errorf("%s: %+v, want %q", tt.name, e, tt.want)
		}
	}
}

func TestReplyWithoutNOrWithAnEqualNumberIsAccepted(t *testing.T) {
	for _, body := range []string{`{"m": [{"t": "rp", "b": "sess-1"}]}`, `{"n": 1.0, "m": [{"b": "sess-1"}]}`,
		`{"n": true, "m": [{"b": "sess-1"}]}`} {
		st := newScripted(t, func(s seen) *reply { return &reply{Status: 200, Body: body} })
		got, err := newClient(t, st.URL, nil).Call("ssession", "make", obj())
		if err != nil || got != "sess-1" {
			t.Fatalf("%s: %v, %v", body, got, err)
		}
	}
}

func TestTimeoutIsPerSocketOperation(t *testing.T) {
	block := make(chan struct{})
	st := newScripted(t, func(seen) *reply { <-block; return nil })
	t.Cleanup(func() { close(block) }) // runs before the server's Close, which waits for it
	c := newClient(t, st.URL, func(c *Config) { c.Timeout = 0.3 })
	start := time.Now()
	_, err := c.Call(Channel, "make", obj())
	if e := boxErr(t, err, KindBox); e.Msg != "transport failure: timed out" {
		t.Fatalf("%q", e.Msg)
	}
	if d := time.Since(start); d < 250*time.Millisecond || d > 3*time.Second {
		t.Fatalf("took %v", d)
	}
}

func TestInvalidateHandlesIsBoundaryAware(t *testing.T) {
	c := newClient(t, "http://127.0.0.1:1", nil)
	for i, ord := range []string{"station:|slot:/A", "station:|slot:/A/B", "station:|slot:/AB", "station:|slot:/AB/C"} {
		c.SetHandle(ord, itoa(i))
	}
	c.InvalidateHandles("station:|slot:/A")
	if got := pyjson.Repr(c.Handles()); got != "{'station:|slot:/AB': '2', 'station:|slot:/AB/C': '3'}" {
		t.Fatalf("after /A: %s", got)
	}
	c.InvalidateHandles("station:|slot:/AB/") // a trailing slash is the same boundary
	c.SetHandle("station:", "2")
	c.ClearHandles()
	if c.Handles().Len() != 0 {
		t.Fatalf("after clear: %s", pyjson.Repr(c.Handles()))
	}
}
