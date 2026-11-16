package plan

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Refused is a journaled batch the station explicitly refused, a read-back showing none of it
// applied (bench retro 2026-11-15 delta 4): its tool error and its batch id.
type Refused struct{ Msg, BatchID string }

func (e *Refused) Error() string { return e.Msg }

// PyType is the class the server answers it as.
func (e *Refused) PyType() string { return "ToolError" }

// refused is the Refused error of a batch whose first op failed with why, once its refused
// result record is journaled. It is nil, and the batch stays in-doubt, unless why is the
// station's own error reply (not a transport or protocol failure), the tool can read back its
// targets (Impl.Unapplied), that read-back succeeds and shows nothing applied, and the record
// is written.
func (e *Engine) refused(c Client, impl Impl, args *pyjson.Object, planned *Planned, batchID string, why error) *Refused {
	var be *box.Error
	if !errors.As(why, &be) || !be.Station || impl.Unapplied == nil {
		return nil
	}
	none, err := jsonrpc.Recover(func() (bool, error) { return impl.Unapplied(c, args, planned) })
	if err != nil || !none {
		return nil
	}
	text := e.Ctx.Scrub(be.Msg)
	result := obj("batch_id", batchID, "ts", session.Now(), "phase", "result", "accepted", nil,
		"inverse", []any{}, "verdict", journal.StateRefused, "refused", true, "station_error", text)
	if e.State.Journal.Append(result) != nil {
		return nil // no record: the intent alone leaves the batch in-doubt, and the reply says so
	}
	return &Refused{BatchID: batchID, Msg: fmt.Sprintf(refusedReply, batchID, text)}
}
