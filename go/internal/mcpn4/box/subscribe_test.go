package box

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Subscribe and Unsubscribe (#875) send one component-space callssc each, the ORDs as the
// list box-rt's subscribe and unsubscribe resolve.
func TestSubscribeAndUnsubscribeSendTheOrdList(t *testing.T) {
	st := newScripted(t, func(req seen) *reply {
		switch req.key() {
		case "callssc:sub":
			return &reply{Status: 200, Body: `{"p":"box","m":[{"r":"0","t":"r","b":{"e":[],"h":["1"],"ts":"x"}}]}`}
		case "callssc:unsub":
			return &reply{Status: 200, Body: `{"p":"box","m":[{"r":"0","t":"r","b":null}]}`}
		}
		return nil
	})
	c := newClient(t, st.URL, nil)
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	ords := []string{"station:|slot:/A/P"}
	if err := c.Subscribe(ords); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := c.Unsubscribe(ords); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if countKey(st, "callssc:sub") != 1 || countKey(st, "callssc:unsub") != 1 {
		t.Fatalf("keys %v", st.keys())
	}
	for _, req := range st.requests() {
		if k := req.key(); k == "callssc:sub" || k == "callssc:unsub" {
			m, _ := req.Frame.Get("m")
			b, _ := get(m.([]any)[0], "b")
			arg, _ := get(b, "scarg")
			if pyjson.Repr(arg) != "['station:|slot:/A/P']" {
				t.Errorf("%s scarg %s", k, pyjson.Repr(arg))
			}
		}
	}
}
