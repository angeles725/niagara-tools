package box

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// frameMessages are the "m" entries of a POSTed BOX frame.
func frameMessages(s seen) []any {
	m, _ := s.Frame.Get("m")
	msgs, _ := m.([]any)
	return msgs
}

// batchStation answers every frame with the entries answer builds from the frame's messages.
func batchStation(t *testing.T, answer func(msgs []any) []any) *scriptedStation {
	return newScripted(t, func(s seen) *reply {
		n, _ := s.Frame.Get("n")
		out, _ := pyjson.Marshal(obj("v", "2.3", "p", "box", "n", n, "m", answer(frameMessages(s))), pyjson.Options{})
		return &reply{Status: 200, Body: string(out)}
	})
}

// echo answers each message with its own key, in request order.
func echo(msgs []any) []any {
	var out []any
	for _, m := range msgs {
		r, _ := get(m, "r")
		k, _ := get(m, "k")
		out = append(out, obj("r", r, "t", "rp", "b", k))
	}
	return out
}

func many(n int) []Msg {
	var out []Msg
	for i := range n {
		out = append(out, Msg{Channel: Channel, Key: fmt.Sprintf("k%d", i), Body: obj("i", i)})
	}
	return out
}

func bodies(rs []Reply) string {
	var out []string
	for _, r := range rs {
		if r.Err != nil {
			out = append(out, "ERR:"+r.Err.Error())
		} else {
			out = append(out, pyjson.Str(r.Body))
		}
	}
	return strings.Join(out, " ")
}

func TestCallManySendsOneFrameWithRFromZero(t *testing.T) {
	st := batchStation(t, echo)
	c := newClient(t, st.URL, nil)
	rs, err := c.CallMany(many(3))
	if err != nil || bodies(rs) != "k0 k1 k2" {
		t.Fatalf("replies %q, err %v", bodies(rs), err)
	}
	reqs := st.requests()
	if len(reqs) != 1 || c.Frames() != 1 {
		t.Fatalf("%d requests, Frames() %d; want 1 and 1", len(reqs), c.Frames())
	}
	want := `"m": [{"r": 0, "t": "rt", "c": "ssession", "k": "k0", "b": {"i": 0}}, ` +
		`{"r": 1, "t": "rt", "c": "ssession", "k": "k1", "b": {"i": 1}}, ` +
		`{"r": 2, "t": "rt", "c": "ssession", "k": "k2", "b": {"i": 2}}], "n": 1`
	if !strings.Contains(string(reqs[0].Body), want) {
		t.Fatalf("frame %s", reqs[0].Body)
	}
}

func TestCallManyMatchesRepliesByRWhateverTheirOrder(t *testing.T) {
	st := batchStation(t, func(msgs []any) []any {
		e := echo(msgs)
		return []any{e[2], e[0], e[1]}
	})
	rs, err := newClient(t, st.URL, nil).CallMany(many(3))
	if err != nil || bodies(rs) != "k0 k1 k2" {
		t.Fatalf("replies %q, err %v", bodies(rs), err)
	}
}

func TestCallManyOneFailingEntryIsItsOwnErrorAndNeverFailsTheSiblings(t *testing.T) {
	st := batchStation(t, func(msgs []any) []any {
		e := echo(msgs)
		e[1] = obj("r", 1, "t", "e", "b", obj("m", "no &quot;k1&quot;"))
		return e
	})
	rs, err := newClient(t, st.URL, nil).CallMany(many(3))
	if err != nil || bodies(rs) != `k0 ERR:no "k1" k2` {
		t.Fatalf("replies %q, err %v", bodies(rs), err)
	}
	if e := boxErr(t, rs[1].Err, KindBox); !e.Station || e.Channel != Channel || e.Msg != `no "k1"` {
		t.Fatalf("entry error %+v", e)
	}
}

func TestCallManyRefusesAReplyThatDoesNotAnswerEveryMessageOnce(t *testing.T) {
	for name, answer := range map[string]func(e []any) []any{
		"missing r":   func(e []any) []any { return e[:2] },
		"duplicate r": func(e []any) []any { return []any{e[0], e[0], e[2]} },
		"extra r":     func(e []any) []any { return append(e, obj("r", 3, "t", "rp", "b", "x")) },
		"r not int":   func(e []any) []any { e[1] = obj("r", "1", "t", "rp", "b", "x"); return e },
		"no r":        func(e []any) []any { e[1] = obj("t", "rp", "b", "x"); return e },
		"not object":  func(e []any) []any { e[1] = "x"; return e },
	} {
		st := batchStation(t, func(msgs []any) []any { return answer(echo(msgs)) })
		rs, err := newClient(t, st.URL, nil).CallMany(many(3))
		e := boxErr(t, err, KindBox)
		if rs != nil || !strings.HasPrefix(e.Msg, "malformed BOX reply: ") {
			t.Errorf("%s: replies %v, err %+v", name, rs, e)
		}
	}
}

func TestCallManySplitsALargerInputIntoFramesOfAtMostTheCap(t *testing.T) {
	st := batchStation(t, echo)
	c := newClient(t, st.URL, nil)
	total := 2*MaxFrameMessages + 20
	rs, err := c.CallMany(many(total))
	if err != nil || len(rs) != total || bodies(rs[:2]) != "k0 k1" ||
		pyjson.Str(rs[total-1].Body) != fmt.Sprintf("k%d", total-1) {
		t.Fatalf("%d replies, err %v", len(rs), err)
	}
	var sizes []int
	for _, r := range st.requests() {
		sizes = append(sizes, len(frameMessages(r)))
	}
	if fmt.Sprint(sizes) != fmt.Sprintf("[%d %d 20]", MaxFrameMessages, MaxFrameMessages) || c.Frames() != 3 {
		t.Fatalf("frame sizes %v, Frames() %d", sizes, c.Frames())
	}
	c.FrameCap = 3
	if _, err := c.CallMany(many(7)); err != nil || c.Frames() != 3+3 {
		t.Fatalf("cap 3 over 7 messages: Frames() %d, err %v; want 6", c.Frames(), err)
	}
}

func TestCallManyOfNothingSendsNothing(t *testing.T) {
	st := batchStation(t, echo)
	c := newClient(t, st.URL, nil)
	if rs, err := c.CallMany(nil); err != nil || len(rs) != 0 || len(st.requests()) != 0 || c.Frames() != 0 {
		t.Fatalf("replies %v, err %v, requests %d", rs, err, len(st.requests()))
	}
}
