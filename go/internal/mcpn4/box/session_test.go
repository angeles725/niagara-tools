package box

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestOpenSendsPythonFramesAndHeaders(t *testing.T) {
	st := newScripted(t, nil)
	c := newClient(t, st.URL, nil)
	root, err := c.Open()
	if err != nil {
		t.Fatal(err)
	}
	if h, _ := get(root, "h"); h != "2" {
		t.Fatalf("root = %s", pyjson.Repr(root))
	}
	c.Close()
	want := []string{
		`{"p": "box", "v": "2.3", "m": [{"r": 0, "t": "rt", "c": "ssession", "k": "make", "b": {}}], "n": 1}`,
		`{"p": "box", "v": "2.3", "m": [{"r": 0, "t": "rt", "c": "ssession", "k": "makessc", "b": {"id": "sess-1", "scid": "cs1", "scts": "box:ComponentSpaceSessionHandler", "scarg": "station:"}}], "n": 2, "id": "sess-1"}`,
		`{"p": "box", "v": "2.3", "m": [{"r": 0, "t": "rt", "c": "ssession", "k": "callssc", "b": {"id": "sess-1", "scid": "cs1", "sck": "loadRoot", "scarg": null}}], "n": 3, "id": "sess-1"}`,
		`{"p": "box", "v": "2.3", "m": [{"r": 0, "t": "rt", "c": "ssession", "k": "del", "b": {"id": "sess-1"}}], "n": 4, "id": "sess-1"}`,
	}
	reqs := st.requests()
	if len(reqs) != len(want) {
		t.Fatalf("requests = %v", st.keys())
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("admin:secret"))
	for i, r := range reqs {
		if string(r.Body) != want[i] {
			t.Errorf("request %d:\n got  %s\n want %s", i, r.Body, want[i])
		}
		if r.Method != "POST" || r.Path != "/box/" || r.Header.Get("Authorization") != auth ||
			r.Header.Get("Content-Type") != "application/json" ||
			r.Header.Get("Accept-Encoding") != "identity" || r.Header.Get("Connection") != "close" {
			t.Errorf("request %d: %s %s %v", i, r.Method, r.Path, r.Header)
		}
	}
	if c.SID() != nil {
		t.Fatalf("sid after close = %v", c.SID())
	}
}

func TestAuthFailureRaisesOnceWithoutRetry(t *testing.T) {
	st := newScripted(t, func(s seen) *reply { return &reply{Status: 401} })
	var hooked []string
	c := newClient(t, st.URL, func(c *Config) { c.Password = "wrong" })
	c.OnAuthError = func(e *Error) { hooked = append(hooked, e.Msg) }
	_, err := c.Open()
	e := boxErr(t, err, KindAuth)
	for _, needle := range []string{"HTTP 401", "HTTPBasicScheme", "password", "do not retry"} {
		if !strings.Contains(e.Msg, needle) {
			t.Errorf("%q lacks %q", e.Msg, needle)
		}
	}
	if e.Channel != "ssession" || e.Key != "make" || len(st.requests()) != 1 || len(hooked) != 1 {
		t.Fatalf("err %+v, requests %v, hooked %v", e, st.keys(), hooked)
	}
}

func TestAuthFailureAfterMakeSendsNoCleanupDel(t *testing.T) {
	for _, tt := range []struct {
		key  string
		code int
	}{{"makessc", 401}, {"callssc:loadRoot", 403}} {
		st := newScripted(t, func(s seen) *reply {
			if s.key() == tt.key {
				return &reply{Status: tt.code}
			}
			return nil
		})
		c := newClient(t, st.URL, nil)
		_, err := c.Open()
		boxErr(t, err, KindAuth)
		if keys := strings.Join(st.keys(), ","); strings.Contains(keys, "del") || c.SID() != nil {
			t.Fatalf("%s: requests %s, sid %v", tt.key, keys, c.SID())
		}
	}
}

func TestNonAuthFailureAfterMakeClosesTheSession(t *testing.T) {
	for _, key := range []string{"makessc", "callssc:loadRoot"} {
		st := newScripted(t, func(s seen) *reply {
			if s.key() == key {
				return &reply{Status: 500}
			}
			return nil
		})
		c := newClient(t, st.URL, nil)
		_, err := c.Open()
		if e := boxErr(t, err, KindBox); e.Msg != "HTTP 500 from station" {
			t.Fatalf("%s: %q", key, e.Msg)
		}
		if keys := st.keys(); keys[len(keys)-1] != "del" || c.SID() != nil {
			t.Fatalf("%s: requests %v, sid %v", key, keys, c.SID())
		}
	}
}

func TestStationErrorFrameRaisesBoxError(t *testing.T) {
	st := newScripted(t, nil)
	c := newClient(t, st.URL, nil)
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	_, err := c.SSC("noSuchKey", nil)
	if e := boxErr(t, err, KindBox); e.Msg != "unknown key" || e.Key != "callssc" {
		t.Fatalf("%+v", e)
	}
}
