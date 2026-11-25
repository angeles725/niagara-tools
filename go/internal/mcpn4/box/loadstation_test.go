package box

import (
	"fmt"
	"sync"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// loadStation answers loadSlots messages with null and queues one load op per message for the next
// pollchgs, in request order unless reorder says otherwise. An ord in fail gets a "t": "e" entry, an
// ord in silent never gets its op. The op of ord X has handle "h:X" and one child "c".
type loadStation struct {
	mu           sync.Mutex
	queue        []any
	fail, silent map[string]bool
	reorder      func(ops []any) []any
	loadFrames   int
	loadMsgs     int
	frameOrds    [][]string    // the ords of every loadSlots frame that arrived, in order
	stallFrame   int           // the n-th loadSlots frame (1-based) is dropped: no answer, no load op
	failFrame    int           // the n-th loadSlots frame is answered HTTP 500
	block        chan struct{} // stalled requests wait here until the test ends
	*scriptedStation
}

func newLoadStation(t *testing.T) *loadStation {
	ls := &loadStation{fail: map[string]bool{}, silent: map[string]bool{}}
	ls.scriptedStation = newScripted(t, func(s seen) *reply {
		switch s.key() {
		case "callssc:loadSlots":
			return ls.loads(s)
		case "pollchgs":
			ls.mu.Lock()
			defer ls.mu.Unlock()
			events := ls.queue
			ls.queue = nil
			if events == nil {
				events = []any{}
			}
			return boxReply(s, events, false)
		}
		return nil
	})
	return ls
}

// loadOrd is the ord a loadSlots message asks for.
func loadOrd(m any) string {
	b, _ := get(m, "b")
	arg, _ := get(b, "scarg")
	return pyjson.Str(field(arg.(*pyjson.Object), "o"))
}

// stallLoadFrame drops the n-th loadSlots frame: its request gets no answer until the test ends
// (the client times out) and the station never loads its ords.
func (ls *loadStation) stallLoadFrame(t *testing.T, n int) {
	ls.stallFrame, ls.block = n, make(chan struct{})
	t.Cleanup(func() { close(ls.block) }) // runs before the server's Close, which waits for it
}

func (ls *loadStation) loads(s seen) *reply {
	ls.mu.Lock()
	ls.loadFrames++
	nth := ls.loadFrames
	var asked []string
	for _, m := range frameMessages(s) {
		asked = append(asked, loadOrd(m))
	}
	ls.frameOrds = append(ls.frameOrds, asked)
	ls.mu.Unlock()
	if nth == ls.stallFrame {
		<-ls.block
		return &reply{Status: 500}
	}
	if nth == ls.failFrame {
		return &reply{Status: 500}
	}
	ls.mu.Lock()
	defer ls.mu.Unlock()
	var entries, ops []any
	for _, m := range frameMessages(s) {
		ls.loadMsgs++
		r, _ := get(m, "r")
		ord := loadOrd(m)
		if ls.fail[ord] {
			entries = append(entries, obj("r", r, "t", "e", "b", obj("m", "denied "+ord)))
			continue
		}
		entries = append(entries, obj("r", r, "t", "rp", "b", nil))
		if !ls.silent[ord] {
			ops = append(ops, obj("nm", "l", "h", "h:"+ord, "b", obj("nm", "p", "h", "h:"+ord,
				"s", []any{obj("n", "c", "t", "x:"+ord)})))
		}
	}
	if ls.reorder != nil {
		ops = ls.reorder(ops)
	}
	for _, op := range ops {
		ls.queue = append(ls.queue, obj("evs", obj("ops", []any{op})))
	}
	n, _ := s.Frame.Get("n")
	out, _ := pyjson.Marshal(obj("v", "2.3", "p", "box", "n", n, "m", entries), pyjson.Options{})
	return &reply{Status: 200, Body: string(out)}
}

func ordsOf(n int) []string {
	var out []string
	for i := range n {
		out = append(out, fmt.Sprintf("station:|slot:/P%d", i))
	}
	return out
}

func openClient(t *testing.T, ls *loadStation) *Client {
	c := newClient(t, ls.URL, nil)
	if _, err := c.Open(); err != nil {
		t.Fatal(err)
	}
	return c
}

// checkNodes fails unless every result is its ord's own load.
func checkNodes(t *testing.T, ords []string, rs []LoadResult) {
	t.Helper()
	if len(rs) != len(ords) {
		t.Fatalf("%d results for %d ords", len(rs), len(ords))
	}
	for i, ord := range ords {
		if rs[i].Err != nil || pyjson.Str(field(field(rs[i].Nodes, "").(*pyjson.Object), "h")) != "h:"+ord ||
			pyjson.Str(field(field(rs[i].Nodes, "c").(*pyjson.Object), "t")) != "x:"+ord {
			t.Fatalf("result %d for %s: %v, %v", i, ord, rs[i].Nodes, rs[i].Err)
		}
	}
}
