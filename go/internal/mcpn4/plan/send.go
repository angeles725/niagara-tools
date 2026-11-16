package plan

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// sendAll is _send_all: the batch's ops in order; a failure at op k is its partial in-doubt
// error, except an explicit station refusal of op 1 that a read-back shows changed nothing.
func (e *Engine) sendAll(c Client, impl Impl, args *pyjson.Object, planned *Planned, batchID string) ([]any, error) {
	replies := []any{}
	for _, op := range planned.Ops {
		reply, err := send(c, op)
		if err != nil {
			if len(replies) == 0 { // the first op: a station refusal may have changed nothing
				if refused := e.refused(c, impl, args, planned, batchID, err); refused != nil {
					return nil, refused
				}
			}
			return nil, e.inDoubtPartial(impl, planned, batchID, replies, err)
		}
		replies = append(replies, reply)
	}
	return replies, nil
}

// inDoubtPartial is _in_doubt_partial: op k of a multi-op batch failed after ops 0..k-1 were
// accepted. Their replies and inverse and an in-doubt result record go to the journal, so the
// batch never looks completed; a one-op batch keeps the plain in-doubt error (its intent
// without a result says everything).
func (e *Engine) inDoubtPartial(impl Impl, planned *Planned, batchID string, replies []any, why error) error {
	err := inDoubt(batchID, why)
	total, sent := len(planned.Ops), len(replies)
	if total == 1 {
		return err
	}
	head := *planned
	head.Ops = planned.Ops[:sent]
	inverse, ierr := jsonrpc.Recover(func() ([]any, error) { return impl.inverse(&head, replies) })
	shown := "unknown" // an inverse cannot be built from these replies: say so
	if inverse = list(inverse); ierr == nil {
		shown = canonical(inverse)
	}
	done := "no op before it was sent"
	switch {
	case sent == 1:
		done = "op 1 was sent"
	case sent > 1:
		done = fmt.Sprintf("ops 1-%d were sent", sent)
	}
	err.Msg = fmt.Sprintf("%s; op %d of %d failed: %s and accepted by the station (replies: %s), op %d may or may not "+
		"have been applied and no later op was sent. Inverse of the sent ops: %s", err.Msg, sent+1, total, done,
		canonical(replies), sent+1, shown)
	if ierr != nil {
		inverse = []any{}
	}
	if e.State.Journal.Append(obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", replies,
		"inverse", inverse, "verdict", "unverified", "in_doubt", true, "sent", sent, "failed_op", sent)) != nil {
		err.Msg += " (the in-doubt result record could not be journaled)"
	}
	return err
}

// send is _send: an ssc op or one sync op; a removal makes every cached handle stale.
func send(c Client, op any) (any, error) {
	o, _ := op.(*pyjson.Object) // every planned op is a dict
	if key, ok := o.Get("ssc"); ok {
		arg, _ := o.Get("arg")
		return c.SSC(pyjson.Str(key), arg)
	}
	reply, err := c.Sync(op)
	if nm, _ := o.Get("nm"); err == nil && nm == "v" {
		c.ClearHandles()
	}
	return reply, err
}
