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

func (ls *loadStation) loads(s seen) *reply {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.loadFrames++
	var entries, ops []any
	for _, m := range frameMessages(s) {
		ls.loadMsgs++
		r, _ := get(m, "r")
		b, _ := get(m, "b")
		arg, _ := get(b, "scarg")
		ord := pyjson.Str(field(arg.(*pyjson.Object), "o"))
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
