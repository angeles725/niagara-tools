package box

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// TestReplayRecordedFrames is the recorded-frame differential of slice S3: every scenario of
// testdata/replay.jsonl (written by mcp-n4-kit/tests/go_box.py from a Python BoxClient talking
// to the recording FakeStation, until S17 removed both; the file is now frozen) is replayed to
// the Go client. The station answers each request
// with the next recorded reply, every request body must equal the recorded one byte for byte,
// no recorded frame may be left over, and each op result must equal Python's. Nothing is
// masked: the FakeStation session id and handles are deterministic and no request carries a
// nonce, token or timestamp.
func TestReplayRecordedFrames(t *testing.T) {
	f, err := os.Open("testdata/replay.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<24)
	n := 0
	for sc.Scan() {
		v, err := pyjson.Unmarshal(sc.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		s := v.(*pyjson.Object)
		n++
		t.Run(field(s, "name").(string), func(t *testing.T) { replay(t, s) })
	}
	if n < 40 {
		t.Fatalf("replay.jsonl holds %d scenarios, want the full corpus", n)
	}
}

func replay(t *testing.T, s *pyjson.Object) {
	frames := field(s, "frames").([]any)
	var mu sync.Mutex
	next := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		i := next
		next++
		mu.Unlock()
		if i >= len(frames) {
			t.Errorf("request %d beyond the %d recorded frames: %s", i, len(frames), body)
			w.WriteHeader(599)
			return
		}
		fr := frames[i].(*pyjson.Object)
		if want := field(fr, "request").(string); string(body) != want || r.URL.Path != "/box/" {
			t.Errorf("request %d (%s):\n go     %s\n python %s", i, r.URL.Path, body, want)
		}
		status := intField(fr, "status")
		reply := field(fr, "reply").(string)
		w.Header().Set("Content-Length", strconv.Itoa(len(reply)))
		if status == 200 {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(status)
		io.WriteString(w, reply)
	}))
	defer srv.Close()
	c, err := clientFrom(field(s, "client").(*pyjson.Object), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	ops := field(s, "ops").([]any)
	got := runOps(t, c, ops)
	want := field(s, "results").([]any)
	for i := range want {
		if got[i] != want[i].(string) {
			t.Errorf("op %d %s:\n go     %s\n python %s", i, pyjson.Repr(ops[i]), got[i], want[i])
		}
	}
	if next != len(frames) {
		t.Errorf("Go sent %d requests, Python %d", next, len(frames))
	}
}
