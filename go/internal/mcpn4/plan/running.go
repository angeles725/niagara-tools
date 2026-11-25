package plan

import (
	"fmt"
	"sync"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// DefaultReplyBudget is the reply budget the server gives a write (#925): under the 60 s an MCP
// client commonly waits for a request, with room for the reply to reach it.
const DefaultReplyBudget = 45 * time.Second

// runningNote is what a {state: running} reply says the caller does next.
const runningNote = "the write was still running when the reply budget (%s) ran out; it goes on and " +
	"its result is journaled: the next call waits for it, then n4_list_batches shows the batch " +
	"(completed or in-doubt) with its verdict. Do not send the confirmation token again"

// within runs an executing write; past the reply budget it answers {state: running, batch_id} and
// leaves the write to the serve loop's background work, which audits its end. The loop does not
// read the next request before that work ends, so the station session is never used twice at
// once. Without a budget or a background, the write runs to its end first (as before #925).
func (e *Engine) within(name string, args *pyjson.Object, batchID string, budget any,
	run func() (*pyjson.Object, error)) (*pyjson.Object, error) {
	if e.ReplyBudget <= 0 || e.Background == nil {
		return run()
	}
	type ended struct {
		out *pyjson.Object
		err error
	}
	var mu sync.Mutex // late and the send on done: either the reply or the background audits the end
	late, done := false, make(chan ended, 1)
	e.Background.Go(func() {
		out, err := run()
		mu.Lock()
		defer mu.Unlock()
		if late { // no serve loop Recover around it here: a panic in the audit must not end the server
			_, _ = jsonrpc.Recover(func() (*pyjson.Object, error) { return e.finish(name, args, batchID, out, err) })
			return
		}
		done <- ended{out, err}
	})
	timer := time.NewTimer(e.ReplyBudget)
	defer timer.Stop()
	select {
	case r := <-done:
		return r.out, r.err
	case <-timer.C:
	}
	mu.Lock()
	defer mu.Unlock()
	select {
	case r := <-done: // it ended while the budget ran out
		return r.out, r.err
	default:
	}
	late = true
	e.record(name, args, batchID, "running", "reply budget passed; the write goes on in the background")
	return nil, &Running{Reply: obj("state", "running", "batch_id", batchID, "executed", true,
		"write_budget", budget, "note", fmt.Sprintf(runningNote, e.ReplyBudget))}
}

// Running is the reply of a write still running when its reply budget ran out: Handler answers
// it as the tool's result, its audit line already written.
type Running struct{ Reply *pyjson.Object }

func (r *Running) Error() string { return "running" }
