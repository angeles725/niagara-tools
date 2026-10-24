package plan

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// Handler is _make_handler(name): the tool's pipeline plus its audit line.
func (e *Engine) Handler(name string, impl Impl) jsonrpc.Handler {
	return func(args *pyjson.Object) (*pyjson.Object, error) {
		batchID := ""
		out, err := jsonrpc.Recover(func() (*pyjson.Object, error) { // a panic is audited as an error
			out, id, err := e.process(name, impl, args)
			batchID = id
			return out, err
		})
		var te *jsonrpc.ToolError
		var se *safety.Error
		var doubt *InDoubt
		switch {
		case errors.As(err, &doubt): // the station call failed after the intent was journaled
			e.record(name, args, doubt.BatchID, "error", doubt.Msg)
			return nil, ToolError(doubt.Msg)
		case errors.As(err, &te) || errors.As(err, &se):
			e.record(name, args, nil, "refused", PyStr(err))
			return nil, ToolError(PyStr(err))
		case err != nil:
			e.record(name, args, nil, "error", PyType(err)+": "+PyStr(err))
			return nil, err
		}
		logged := true
		if batchID == "" {
			logged = e.record(name, args, nil, "planned", "dry run")
		} else {
			verdict, _ := out.Get("verdict")
			logged = e.record(name, args, batchID, "executed", "verdict="+pyjson.Str(verdict))
		}
		if !logged {
			warnings, _ := out.Get("warnings")
			list, _ := warnings.([]any)
			out.Set("warnings", append(list, "the audit log could not be written"))
		}
		return out, nil
	}
}

// process is _process: the plan and either its dry run or its execution; batchID is "" for a
// dry run.
func (e *Engine) process(name string, impl Impl, args *pyjson.Object) (out *pyjson.Object, batchID string, err error) {
	w, sess := e.State, e.Ctx.Session
	dry := true
	if v, ok := args.Get("dry_run"); ok {
		dry = pyjson.Truthy(v)
	}
	if sess == nil {
		return nil, "", ToolError(safety.ReasonNotConnected + ": call n4_connect first")
	}
	if !sess.IdentityVerified {
		return nil, "", ToolError(safety.ReasonIdentity + ": reconnect with n4_connect and pass " +
			"expected_station=<the station's stationName> before any write")
	}
	tierBlock := e.Ctx.TierBlock(sess) // METHODOLOGY section 5: a dry run is still allowed
	if tierBlock != "" && !dry {
		return nil, "", ToolError(tierBlock)
	}
	if w.WritesExecuted >= w.MaxWrites {
		return nil, "", ToolError(fmt.Sprintf("%s: %d writes already executed by this server process "+
			"(--max-writes; a reconnect does not reset it)", safety.ReasonBudget, w.MaxWrites))
	}
	ords := impl.ScopeOrds(args)
	for _, o := range ords {
		if err := w.Scope.Check(o); err != nil {
			return nil, "", err
		}
	}
	if len(ords) == 0 { // whole-station or journal-driven: at least one scope must be configured
		if err := w.Scope.RequireAny(); err != nil {
			return nil, "", err
		}
	}
	client, ok := sess.Client.(Client)
	if !ok {
		return nil, "", &PyError{Type: "AttributeError", Msg: "the station client cannot write"}
	}
	planned, err := impl.Plan(client, args)
	if err != nil {
		return nil, "", err
	}
	plan, planHash, err := Hashed(name, planned, sess)
	if err != nil {
		return nil, "", err
	}
	if dry {
		return e.dryRun(name, args, plan, planHash, tierBlock)
	}
	if err := w.checkStateFiles(); err != nil { // a loose file fails before the token and any send
		return nil, "", err
	}
	tok, _ := args.Get("confirmation_token")
	if err := w.Tokens.Consume(name, args, planHash, tok); err != nil {
		return nil, "", err
	}
	batchID = w.NewBatchID()
	targets := planned.Targets
	if targets == nil {
		targets = pyjson.NewObject()
	}
	intent := obj("batch_id", batchID, "ts", session.Now(), "tool", name, "ops", list(planned.Ops),
		"inverse_plan", list(planned.Inverse), "station_name", sess.StationName, "phase", "intent", "targets", targets)
	if w.Journal.Append(intent) != nil { // write-ahead: no intent on disk, no op on the wire
		return nil, "", ToolError("nothing was sent: the journal intent could not be written (batch " + batchID +
			"); the confirmation token is spent, run the dry run again")
	}
	w.WritesExecuted++ // counted once sent, whatever the station answers
	// From here on the batch exists: execute's only error is its in-doubt one, and a panic
	// (Python's unexpected exception) leaves the batch in-doubt too.
	out, err = jsonrpc.Recover(func() (*pyjson.Object, error) { return e.execute(client, impl, args, planned, batchID) })
	var p *jsonrpc.Panic
	if errors.As(err, &p) {
		err = inDoubt(batchID, err)
	}
	return out, batchID, err
}
