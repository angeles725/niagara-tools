package box

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// LoadTypes asks the station's "reg" channel (BRegistryChannel.loadTypes) for the type and its
// dependencies with their frozen contracts (ec: true), and returns the reply body as it came.
func TestLoadTypesSendsOneRegistryFrameWithContracts(t *testing.T) {
	var sent *pyjson.Object
	st := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		v, err := pyjson.Unmarshal(body)
		if err != nil {
			t.Errorf("request is not JSON: %v", err)
		}
		sent = v.(*pyjson.Object)
		n, _ := sent.Get("n")
		w.Write([]byte(`{"p":"box","v":"2.3","n":` + n.(pyjson.Number).String() +
			`,"m":[{"r":0,"t":"rt","c":"reg","k":"loadTypes","b":{"a:B":{"c":[],"p":"a:A"}}}]}`))
	}))
	t.Cleanup(st.Close)
	c := newClient(t, st.URL, nil)
	got, err := c.LoadTypes([]string{"a:B"})
	if err != nil {
		t.Fatal(err)
	}
	msgs, _ := sent.Get("m")
	msg := msgs.([]any)[0].(*pyjson.Object)
	frame, _ := pyjson.Marshal(msg, pyjson.Options{})
	if string(frame) != `{"r": 0, "t": "rt", "c": "reg", "k": "loadTypes", "b": {"t": ["a:B"], "ec": true}}` {
		t.Errorf("frame = %s", frame)
	}
	reply, _ := pyjson.Marshal(got, pyjson.Options{})
	if string(reply) != `{"a:B": {"c": [], "p": "a:A"}}` {
		t.Errorf("reply = %s", reply)
	}
}

// Types do not change within a session: a request whose specs are all cached (requested before or
// sent as a dependency) is answered without a second registry call.
func TestLoadTypesAnswersCachedSpecsWithoutASecondCall(t *testing.T) {
	calls := 0
	st := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		v, _ := pyjson.Unmarshal(body)
		n, _ := v.(*pyjson.Object).Get("n")
		calls++
		w.Write([]byte(`{"p":"box","v":"2.3","n":` + n.(pyjson.Number).String() +
			`,"m":[{"r":0,"t":"rt","c":"reg","k":"loadTypes","b":{"a:B":{"c":[],"p":"a:A"},"a:A":{}}}]}`))
	}))
	t.Cleanup(st.Close)
	c := newClient(t, st.URL, nil)
	if _, err := c.LoadTypes([]string{"a:B"}); err != nil {
		t.Fatal(err)
	}
	got, err := c.LoadTypes([]string{"a:A", "a:B"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
	reply, _ := pyjson.Marshal(got, pyjson.Options{})
	if string(reply) != `{"a:B": {"c": [], "p": "a:A"}, "a:A": {}}` {
		t.Errorf("reply = %s", reply)
	}
}

// A reply that is not an object, or a failed call, is never cached: the next request for the same
// spec asks the station again instead of answering from a half-filled cache.
func TestLoadTypesCachesNeitherAMalformedReplyNorAFailedCall(t *testing.T) {
	calls := 0
	bodies := []string{`[]`, ``, `{"a:B":{"c":[],"p":"a:A"},"a:A":{}}`}
	st := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		v, _ := pyjson.Unmarshal(body)
		n, _ := v.(*pyjson.Object).Get("n")
		b := bodies[calls]
		calls++
		if b == "" {
			w.WriteHeader(599)
			return
		}
		w.Write([]byte(`{"p":"box","v":"2.3","n":` + n.(pyjson.Number).String() +
			`,"m":[{"r":0,"t":"rt","c":"reg","k":"loadTypes","b":` + b + `}]}`))
	}))
	t.Cleanup(st.Close)
	c := newClient(t, st.URL, nil)
	got, err := c.LoadTypes([]string{"a:B"})
	if _, isObj := got.(*pyjson.Object); err != nil || isObj {
		t.Fatalf("malformed reply: got %v, err %v; want it returned as it came", got, err)
	}
	if _, err := c.LoadTypes([]string{"a:B"}); err == nil {
		t.Fatal("failed call: err = nil")
	}
	got, err = c.LoadTypes([]string{"a:B"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3 (nothing cached before the good reply)", calls)
	}
	reply, _ := pyjson.Marshal(got, pyjson.Options{})
	if string(reply) != `{"a:B": {"c": [], "p": "a:A"}, "a:A": {}}` {
		t.Errorf("reply = %s", reply)
	}
}
