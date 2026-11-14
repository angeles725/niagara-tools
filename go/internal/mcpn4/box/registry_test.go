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
