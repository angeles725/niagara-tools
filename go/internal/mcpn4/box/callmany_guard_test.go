package box

import (
	"fmt"
	"strings"
	"testing"
)

func TestCallManyKeepsCallsGuards(t *testing.T) {
	pad := strings.Repeat("x", 4096)
	deep := strings.Repeat("[", MaxReplyNesting+1) + strings.Repeat("]", MaxReplyNesting+1)
	for _, tt := range []struct {
		name, body, want string
		limit            int
	}{
		{"oversize", `{"n": 1, "m": [{"r": 0, "t": "rp", "b": "` + pad + `"}]}`,
			"BOX reply larger than 1024 bytes: refused, not parsed in part", 1024},
		{"nesting", `{"n": 1, "m": [{"r": 0, "t": "rp", "b": ` + deep + `}]}`,
			"malformed BOX reply: nested deeper than 1000 levels", 1 << 20},
		{"seq", `{"n": 9, "m": [{"r": 0, "t": "rp", "b": 1}]}`, "reply seq 9 does not match request seq 1", 1024},
		{"no list", `{"n": 1}`, "malformed reply: missing or empty message list", 1024},
	} {
		st := newScripted(t, func(seen) *reply { return &reply{Status: 200, Body: tt.body} })
		c := newClient(t, st.URL, func(c *Config) { c.MaxReplyBytes = tt.limit })
		rs, err := c.CallMany(many(1))
		if e := boxErr(t, err, KindBox); rs != nil || e.Msg != tt.want {
			t.Errorf("%s: replies %v, err %q, want %q", tt.name, rs, e.Msg, tt.want)
		}
	}
}

func TestCallManyAuthFailureIsAnAuthErrorReportedOnce(t *testing.T) {
	st := newScripted(t, func(seen) *reply { return &reply{Status: 401} })
	c := newClient(t, st.URL, nil)
	var latched []*Error
	c.OnAuthError = func(e *Error) { latched = append(latched, e) }
	rs, err := c.CallMany(many(2))
	if e := boxErr(t, err, KindAuth); rs != nil || e.Msg != AuthMessage(401) || len(latched) != 1 || len(st.requests()) != 1 {
		t.Fatalf("replies %v, err %+v, latched %d, requests %d", rs, e, len(latched), len(st.requests()))
	}
}

func TestSessionLossAnswersEveryEntryWithTheStationError(t *testing.T) {
	st := batchStation(t, func(msgs []any) []any {
		var out []any
		for i := range msgs {
			out = append(out, obj("r", i, "t", "e", "b", obj("m", "session expired")))
		}
		return out
	})
	rs, err := newClient(t, st.URL, nil).CallMany(many(2))
	if err != nil || bodies(rs) != "ERR:session expired ERR:session expired" {
		t.Fatalf("replies %q, err %v", bodies(rs), err)
	}
}

func TestCallStillCountsOneFramePerCall(t *testing.T) {
	st := newScripted(t, nil)
	c := newClient(t, st.URL, nil)
	if got, err := c.Call(Channel, "make", obj()); err != nil || got != "sess-1" || c.Frames() != 1 {
		t.Fatalf("Call = %v, %v, Frames() %d", got, err, c.Frames())
	}
}

func TestSSCManyIsOneCallsscFramePerCapOfArgs(t *testing.T) {
	st := batchStation(t, echo)
	c := newClient(t, st.URL, nil)
	c.sid = "sess-1"
	rs, err := c.SSCMany("impliedTags", []any{"h1", "h2"})
	if err != nil || bodies(rs) != "callssc callssc" || c.Frames() != 1 {
		t.Fatalf("replies %q, err %v, Frames() %d", bodies(rs), err, c.Frames())
	}
	want := `"b": {"id": "sess-1", "scid": "cs1", "sck": "impliedTags", "scarg": "h2"}`
	if !strings.Contains(string(st.requests()[0].Body), want) {
		t.Fatalf("frame %s", st.requests()[0].Body)
	}
}

// OnFrame sees each frame's seq, message count and first key; MCP_N4_BOX_TRACE=1 installs the stderr
// trace line on a new client, and without it there is none.
func TestFrameHookAndTraceEnv(t *testing.T) {
	st := batchStation(t, echo)
	c := newClient(t, st.URL, nil)
	var got []string
	c.OnFrame = func(seq, n int, key string) { got = append(got, fmt.Sprintf("%d %d %s", seq, n, key)) }
	if _, err := c.CallMany(many(3)); err != nil || fmt.Sprint(got) != "[1 3 k0]" {
		t.Fatalf("hook saw %v, err %v", got, err)
	}
	t.Setenv(TraceEnv, "")
	if frameTrace() != nil {
		t.Fatal("trace installed without the env")
	}
	t.Setenv(TraceEnv, "1")
	if frameTrace() == nil {
		t.Fatal("no trace with the env set")
	}
}
